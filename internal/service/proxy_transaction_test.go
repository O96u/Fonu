package service

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/proxy"
)

func TestProxyMutationRollsBackMissingCertificate(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir,"fonu.db"))
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	if err := db.Migrate(ctx,conn,filepath.Join("..","..","migrations")); err != nil { t.Fatal(err) }
	cfg := config.Config{DataDir:dir,NginxBin:"fonu-missing-nginx-binary",NginxPIDFile:filepath.Join(dir,"nginx.pid"),NginxMimeTypes:filepath.Join(dir,"mime.types")}
	store := proxy.NewStore(conn)
	svc := NewProxyService(cfg,conn,store,certificate.NewStore(conn),nil,nginx.NewManager(cfg,slog.Default()))
	rule, err := svc.Create(ctx,proxy.CreateInput{Upstream:"http://127.0.0.1:3000",Hosts:[]string{"app.example.com"},ListenPort:8080,ListenIPv4:true,Enabled:true})
	if err != nil { t.Fatal(err) }
	if rule.EntryID == nil { t.Fatal("missing entry") }
	on := true
	if _, err := svc.UpdateEntry(ctx,*rule.EntryID,proxy.EntryUpdateInput{HTTPSEnabled:&on}); err == nil { t.Fatal("expected certificate validation error") }
	entry, err := store.GetEntry(ctx,*rule.EntryID)
	if err != nil || entry.HTTPSEnabled { t.Fatalf("entry not rolled back: %v",err) }
	got, err := store.Get(ctx,rule.ID)
	if err != nil || got.HTTPSEnabled || !got.Enabled { t.Fatalf("rule not rolled back: %v",err) }
}
