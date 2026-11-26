package certificate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/secret"
)

func TestMigrateAndPersistDeploymentSecrets(t *testing.T) {
	ctx:=context.Background()
	conn,err:=db.Open(filepath.Join(t.TempDir(),"fonu.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
	if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
	box,err:=secret.NewBox("test deployment key");if err!=nil{t.Fatal(err)}
	s:=NewStore(conn,box)
	if _,err:=s.Upsert(ctx,"example.com",false,"","imported","cert.pem","key.pem",time.Now().Add(time.Hour),"valid","");err!=nil{t.Fatal(err)}
	cfg:=ExportConfig{SSH:SSHExportConfig{Host:"example.com",Password:"test-password",PrivateKey:"test-private-key",PrivateKeyPassphrase:"test-passphrase",ProxyPassword:"test-proxy-password"}}
	raw,err:=cfg.Marshal();if err!=nil{t.Fatal(err)}
	if _,err:=conn.Exec("UPDATE certificates SET export_json = ? WHERE domain = 'example.com'",raw);err!=nil{t.Fatal(err)}
	if err:=s.MigrateExportSecrets(ctx);err!=nil{t.Fatal(err)}
	assertEncrypted:=func()string{
		t.Helper()
		var stored string
		if err:=conn.QueryRow("SELECT export_json FROM certificates WHERE domain = 'example.com'").Scan(&stored);err!=nil{t.Fatal(err)}
		for _,v:=range []string{cfg.SSH.Password,cfg.SSH.PrivateKey,cfg.SSH.PrivateKeyPassphrase,cfg.SSH.ProxyPassword}{if strings.Contains(stored,v){t.Fatal("plaintext deployment credential stored")}}
		if !strings.Contains(stored,"ssh_secrets_encrypted"){t.Fatal("missing encrypted envelope")}
		rec,err:=s.GetByDomain(ctx,"example.com");if err!=nil{t.Fatal(err)}
		if extractDeploymentSecrets(rec.Export)!=extractDeploymentSecrets(cfg){t.Fatal("deployment credential roundtrip failed")}
		return stored
	}
	first:=assertEncrypted()
	if err:=s.MigrateExportSecrets(ctx);err!=nil{t.Fatal(err)}
	if first!=assertEncrypted(){t.Fatal("migration not idempotent")}
	if err:=s.SetExport(ctx,"example.com",cfg);err!=nil{t.Fatal(err)}
	assertEncrypted()
	if err:=s.UpdateMeta(ctx,"example.com","new name",cfg);err!=nil{t.Fatal(err)}
	assertEncrypted()
	wrongBox,err:=secret.NewBox("wrong key");if err!=nil{t.Fatal(err)}
	if _,err:=NewStore(conn,wrongBox).GetByDomain(ctx,"example.com");err==nil{t.Fatal("wrong key accepted")}
	if err:=NewStore(conn).SetExport(ctx,"example.com",cfg);err==nil{t.Fatal("plaintext write accepted without encryption key")}
}
