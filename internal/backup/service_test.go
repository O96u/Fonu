package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/secret"
)

func TestRestoreStagesDatabaseUntilNextStart(t *testing.T) {
	ctx := context.Background()
	srcDir := t.TempDir()
	srcDB := filepath.Join(srcDir, "fonu.db")
	conn, err := db.Open(srcDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`CREATE TABLE kept (id INTEGER); INSERT INTO kept VALUES (7)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(srcDB)
	if err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	live := filepath.Join(dataDir, "fonu.db")
	if err := os.WriteFile(live, []byte("still-here"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := writeArchive(archive, map[string][]byte{
		"fonu.db":        payload,
		"certs/a.crt":    []byte("cert"),
		"../outside.txt": []byte("nope"),
	}); err != nil {
		t.Fatal(err)
	}

	svc := New(dataDir)
	if err := svc.Restore(ctx, archive); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "still-here" {
		t.Fatalf("live database was overwritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dataDir), "evil")); err == nil {
		t.Fatal("restore wrote outside the data directory")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dataDir), "outside.txt")); err == nil {
		t.Fatal("restore wrote outside the data directory")
	}

	if err := ApplyPending(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := db.QuickCheck(live); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, pendingDirName)); !os.IsNotExist(err) {
		t.Fatalf("pending restore still present: %v", err)
	}
	cert, err := os.ReadFile(filepath.Join(dataDir, "certs", "a.crt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(cert) != "cert" {
		t.Fatalf("cert = %q", cert)
	}
}

func TestRestoreRejectsBrokenDatabase(t *testing.T) {
	dataDir := t.TempDir()
	live := filepath.Join(dataDir, "fonu.db")
	if err := os.WriteFile(live, []byte("still-here"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "broken.tar.gz")
	if err := writeArchive(archive, map[string][]byte{"fonu.db": []byte("not a database")}); err != nil {
		t.Fatal(err)
	}
	if err := New(dataDir).Restore(context.Background(), archive); err == nil {
		t.Fatal("expected broken database to be rejected")
	}
	got, err := os.ReadFile(live)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "still-here" {
		t.Fatalf("live database changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dataDir, pendingDirName)); !os.IsNotExist(err) {
		t.Fatalf("pending dir left behind: %v", err)
	}
}

func TestRestoreRejectsPathEscape(t *testing.T) {
	dataDir := t.TempDir()
	srcDir := t.TempDir()
	srcDB := filepath.Join(srcDir, "fonu.db")
	conn, err := db.Open(srcDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`CREATE TABLE kept (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(srcDB)
	if err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "escape.tar.gz")
	if err := writeArchive(archive, map[string][]byte{
		"fonu.db":          payload,
		"certs/../../evil": []byte("nope"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := New(dataDir).Restore(context.Background(), archive); err == nil {
		t.Fatal("expected path escape to be rejected")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dataDir), "evil")); err == nil {
		t.Fatal("restore wrote outside the data directory")
	}
}

func TestExportSnapshotsOpenDatabase(t *testing.T) {
	dataDir := t.TempDir()
	conn, err := db.Open(filepath.Join(dataDir, "fonu.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`CREATE TABLE kept (id INTEGER); INSERT INTO kept VALUES (7)`); err != nil {
		t.Fatal(err)
	}

	archive, err := New(dataDir).Export(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(archive)

	outDir := t.TempDir()
	if err := New(outDir).Restore(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	if err := ApplyPending(outDir); err != nil {
		t.Fatal(err)
	}
	restored, err := db.Open(filepath.Join(outDir, "fonu.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	var id int
	if err := restored.QueryRow(`SELECT id FROM kept`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if id != 7 {
		t.Fatalf("id = %d", id)
	}
}

func writeArchive(path string, files map[string][]byte) error {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(body)),
		}); err != nil {
			return err
		}
		if _, err := tw.Write(body); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func TestRestoreRollsBackAllMoves(t *testing.T) {
	dir := t.TempDir()
	pending := filepath.Join(dir, pendingDirName)
	if err := os.Mkdir(pending, 0700); err != nil { t.Fatal(err) }
	for _, name := range []string{"fonu.db", "certs", "nginx", ".secret_key"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("old-"+name), 0600); err != nil { t.Fatal(err) }
		if err := os.WriteFile(filepath.Join(pending, name), []byte("new-"+name), 0600); err != nil { t.Fatal(err) }
	}
	err := swapRestoreFiles(dir, pending, func(from, to string) error {
		if from == filepath.Join(pending, "nginx") { return os.ErrPermission }
		return os.Rename(from, to)
	})
	if err == nil { t.Fatal("expected restore failure") }
	for _, name := range []string{"fonu.db", "certs", "nginx", ".secret_key"} {
		got, err := os.ReadFile(filepath.Join(dir, name)); if err != nil || string(got) != "old-"+name { t.Fatalf("live %s: %q %v", name, got, err) }
		got, err = os.ReadFile(filepath.Join(pending, name)); if err != nil || string(got) != "new-"+name { t.Fatalf("staged %s: %q %v", name, got, err) }
	}
}

func TestRecoverInterruptedRestore(t *testing.T) {
	dir := t.TempDir()
	pending := filepath.Join(dir, pendingDirName)
	rollback := filepath.Join(dir, restoreRollbackName)
	for _, p := range []string{pending, rollback} { if err := os.Mkdir(p,0700); err != nil { t.Fatal(err) } }
	live := filepath.Join(dir, "fonu.db")
	staged := filepath.Join(pending, "fonu.db")
	old := filepath.Join(rollback, "fonu.db")
	if err := os.WriteFile(live, []byte("old"),0600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(staged, []byte("new"),0600); err != nil { t.Fatal(err) }
	j := restoreJournal{Moves: []restoreMove{{From: live, To: old},{From: staged,To:live}}}
	if err := saveRestoreJournal(dir,j); err != nil { t.Fatal(err) }
	if err := os.Rename(live,old); err != nil { t.Fatal(err) }
	if err := os.Rename(staged,live); err != nil { t.Fatal(err) }
	if err := recoverRestore(dir); err != nil { t.Fatal(err) }
	for p,want := range map[string]string{live:"old",staged:"new"} {
		got,err := os.ReadFile(p); if err != nil || string(got)!=want { t.Fatalf("%s: %q %v",p,got,err) }
	}
	if err := recoverRestore(dir); err != nil { t.Fatal(err) }
}

func TestRejectAliasedArchivePath(t *testing.T) {
	if _,err := safeJoin(t.TempDir(),"certs/../fonu.db"); err == nil { t.Fatal("accepted aliased database path") }
}

func TestRecoverEmptyRollbackDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir,restoreRollbackName)
	if err := os.Mkdir(path,0700); err != nil { t.Fatal(err) }
	if err := recoverRestore(dir); err != nil { t.Fatal(err) }
	if _,err := os.Stat(path); !os.IsNotExist(err) { t.Fatal("orphan directory remains") }
}

func TestRestoreRejectsMismatchedCredentialKey(t *testing.T){
 ctx:=context.Background()
 dir:=t.TempDir()
 conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
 key:="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
 wrong:="abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
 box,err:=secret.NewBox(key);if err!=nil{t.Fatal(err)}
 enc,err:=box.Encrypt("test-token");if err!=nil{t.Fatal(err)}
 if _,err:=conn.Exec("CREATE TABLE settings(key TEXT PRIMARY KEY,value TEXT)");err!=nil{t.Fatal(err)}
 if _,err:=conn.Exec("INSERT INTO settings VALUES('frp_auth_token',?)",enc);err!=nil{t.Fatal(err)}
 if err:=conn.Close();err!=nil{t.Fatal(err)}
 raw,err:=os.ReadFile(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
 archive:=filepath.Join(t.TempDir(),"backup.tar.gz")
 target:=t.TempDir()
 if err:=writeArchive(archive,map[string][]byte{"fonu.db":raw,secret.KeyFileName:[]byte(wrong)});err!=nil{t.Fatal(err)}
 if err:=New(target).Restore(ctx,archive);err==nil{t.Fatal("wrong key accepted")}
 if _,err:=os.Stat(filepath.Join(target,pendingDirName));!os.IsNotExist(err){t.Fatal("invalid backup published")}
 if err:=writeArchive(archive,map[string][]byte{"fonu.db":raw,secret.KeyFileName:[]byte(key)});err!=nil{t.Fatal(err)}
 if err:=New(target).Restore(ctx,archive);err!=nil{t.Fatal(err)}
 if err:=ApplyPending(target);err!=nil{t.Fatal(err)}
}
