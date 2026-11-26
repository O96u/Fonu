package frp

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

func TestApplyVerificationFailureRestoresSettingsAndFile(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(dir, "fonu.db"))
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	if err := db.Migrate(ctx, conn, filepath.Join("..","..","migrations")); err != nil { t.Fatal(err) }
	box, err := secret.NewBox("test-secret-key-32bytes-long!!!")
	if err != nil { t.Fatal(err) }
	store := NewStore(settings.NewStore(conn), box)
	if err := store.Save(ctx, SaveInput{ServerAddr:"old.example.com", ServerPort:7000, AuthToken:"old-token"}, false); err != nil { t.Fatal(err) }
	bin := filepath.Join(dir,"frpc")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil { t.Fatal(err) }
	cfg := config.Config{DataDir:dir, FrpcBin:bin, FrpPIDFile:filepath.Join(dir,"frp","frpc.pid"), NginxDefaultHTTPPort:80, NginxDefaultHTTPSPort:443}
	m := NewManager(cfg, store, slog.Default())
	if err := m.EnsureDirs(); err != nil { t.Fatal(err) }
	if err := os.WriteFile(cfg.FrpConfigPath(), []byte("old config"), 0o600); err != nil { t.Fatal(err) }
	_, err = m.Apply(ctx, SaveInput{Enabled:true, ServerAddr:"new.example.com", ServerPort:7000, AuthToken:"new-token", CustomDomains:[]string{"nas.example.com"}})
	if err == nil { t.Fatal("expected verify failure") }
	loaded, token, err := store.LoadRuntime(ctx)
	if err != nil || loaded.Enabled || loaded.ServerAddr != "old.example.com" || token != "old-token" { t.Fatalf("settings not restored: %+v, %v", loaded, err) }
	content, err := os.ReadFile(cfg.FrpConfigPath())
	if err != nil || string(content) != "old config" { t.Fatalf("file changed: %s, %v",content,err) }
}

func TestSessionStatusIgnoresOldLogAndDisconnect(t *testing.T) {
	ok, _, states := sessionStatus([]string{"login to server success", "[fonu-tcp-ssh] start proxy success",sessionMarker})
	if ok || len(states) != 0 { t.Fatal("stale session accepted") }
	ok, _, states = sessionStatus([]string{sessionMarker,"login to server success","[fonu-tcp-ssh] start proxy success","[fonu-udp-dns] start error: port used"})
	if !ok || states["fonu-tcp-ssh"] != "registered" || states["fonu-udp-dns"] != "failed" { t.Fatalf("unexpected states %v",states) }
	ok, _, states = sessionStatus([]string{sessionMarker,"login to server success","[fonu-tcp-ssh] start proxy success","heartbeat timeout"})
	if ok || len(states) != 0 { t.Fatal("disconnect not cleared") }
}

func TestPrivateConfigPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(),"frpc.toml")
	if err := writePrivateConfig(path,[]byte("secret")); err != nil { t.Fatal(err) }
	info, err := os.Stat(path)
	if err != nil { t.Fatal(err) }
	if info.Mode().Perm() != 0o600 { t.Fatalf("unexpected mode %o",info.Mode().Perm()) }
}

func TestBoundedTail(t *testing.T) {
	path := filepath.Join(t.TempDir(),"frpc.log")
	if err := os.WriteFile(path, []byte(strings.Repeat("old line\n", 150000)+"last one\nlast two\n"),0o600); err != nil { t.Fatal(err) }
	lines, err := ReadLogTail(path,2)
	if err != nil || len(lines) != 2 || lines[0] != "last one" || lines[1] != "last two" { t.Fatalf("unexpected tail %v, %v",lines,err) }
}

func TestRejectInvalidFRPInputs(t *testing.T) {
	if err := validateSaveInput(SaveInput{ServerAddr:"host:7000"},false); err == nil { t.Fatal("embedded port accepted") }
	if err := validateSaveInput(SaveInput{CustomDomains:[]string{"https://nas.example.com"}},false); err == nil { t.Fatal("URL accepted as domain") }
	if _, err := prepareTCPProxies([]TCPProxy{{Name:"bad",Protocol:"http"}}); err == nil { t.Fatal("invalid tunnel protocol accepted") }
}

func TestApplyStartFailureRollsBackReplacement(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(dir, "fonu.db"))
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	if err := db.Migrate(ctx, conn, filepath.Join("..","..","migrations")); err != nil { t.Fatal(err) }
	box, err := secret.NewBox("test-secret-key-32bytes-long!!!")
	if err != nil { t.Fatal(err) }
	store := NewStore(settings.NewStore(conn), box)
	if err := store.Save(ctx, SaveInput{ServerAddr:"old.example.com", ServerPort:7000, AuthToken:"old-token"}, false); err != nil { t.Fatal(err) }
	bin := filepath.Join(dir,"frpc")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = verify ]; then exit 0; fi\nexit 1\n"), 0o700); err != nil { t.Fatal(err) }
	cfg := config.Config{DataDir:dir, FrpcBin:bin, FrpPIDFile:filepath.Join(dir,"frp","frpc.pid"), NginxDefaultHTTPPort:80, NginxDefaultHTTPSPort:443}
	m := NewManager(cfg, store, slog.Default())
	if err := m.EnsureDirs(); err != nil { t.Fatal(err) }
	if err := os.WriteFile(cfg.FrpConfigPath(), []byte("old config"), 0o600); err != nil { t.Fatal(err) }
	_, err = m.Apply(ctx, SaveInput{Enabled:true, ServerAddr:"new.example.com", ServerPort:7000, AuthToken:"new-token", CustomDomains:[]string{"nas.example.com"}})
	if err == nil { t.Fatal("expected start failure") }
	loaded, token, err := store.LoadRuntime(ctx)
	if err != nil || loaded.Enabled || loaded.ServerAddr != "old.example.com" || token != "old-token" { t.Fatalf("settings not restored: %+v, %v", loaded, err) }
	content, err := os.ReadFile(cfg.FrpConfigPath())
	if err != nil || string(content) != "old config" { t.Fatalf("file not restored: %s, %v",content,err) }
	if _, err := os.Stat(cfg.FrpPIDFile); !os.IsNotExist(err) { t.Fatalf("stale PID file: %v",err) }
}

func TestCommandOutputBounded(t *testing.T){
	var output commandOutput
	raw:=[]byte(strings.Repeat("x",maxCommandOutput*4))
	n,err:=output.Write(raw)
	if err!=nil||n!=len(raw)||output.Len()!=maxCommandOutput{t.Fatalf("output len=%d n=%d err=%v",output.Len(),n,err)}
	n,err=output.Write(raw)
	if err!=nil||n!=len(raw)||output.Len()!=maxCommandOutput{t.Fatal("output grew beyond limit")}
}
