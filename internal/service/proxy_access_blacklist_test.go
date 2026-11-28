package service

import (
	"context"
	"log/slog"
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/proxy"
)

func TestAccessBlacklistValidation(t *testing.T) {
	for _, invalid := range []string{"", "1.2.3.4/24", "127.0.0.1", "::", "fe80::1%eth0", "1.2.3.4; deny all;"} {
		if _,err:=accessBlacklistIP(invalid);err==nil {t.Fatalf("accepted %q",invalid)}
	}
	ip:=netip.MustParseAddr("192.0.2.1")
	exact,covered:=blacklistCoverage([]string{"192.0.2.1/32","192.0.2.0/24"},ip)
	if !exact||!covered {t.Fatal("expected both exact and network coverage")}
	r:=proxy.Rule{ID:1,Domain:"a.example.com",ListenPort:8080,Enabled:true}
	if _,err:=accessBlacklistRule([]proxy.Rule{r,r},"a.example.com",8080);err==nil {t.Fatal("ambiguous rule accepted")}
	if _,err:=accessBlacklistRule([]proxy.Rule{r},"a.example.com",80);err==nil {t.Fatal("wrong port accepted")}
	r.NginxMode="custom"
	if _,err:=accessBlacklistRule([]proxy.Rule{r},"a.example.com",8080);err==nil {t.Fatal("custom nginx accepted")}
}

func TestAccessBlacklistMutationPreservesOtherRules(t *testing.T) {
	ctx:=context.Background()
	dir:=t.TempDir()
	conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
	defer conn.Close()
	if err=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
	cfg:=config.Config{DataDir:dir,NginxBin:"fonu-missing-nginx-binary",NginxPIDFile:filepath.Join(dir,"nginx.pid"),NginxMimeTypes:filepath.Join(dir,"mime.types")}
	store:=proxy.NewStore(conn)
	svc:=NewProxyService(cfg,conn,store,certificate.NewStore(conn),nil,nginx.NewManager(cfg,slog.Default()))
	a,err:=svc.Create(ctx,proxy.CreateInput{Upstream:"http://127.0.0.1:3000",Hosts:[]string{"a.example.com"},ListenPort:8080,ListenIPv4:true,Enabled:true,Security:proxy.SecurityInput{IPBlacklist:[]string{"198.51.100.0/24"}}});if err!=nil{t.Fatal(err)}
	b,err:=svc.Create(ctx,proxy.CreateInput{Upstream:"http://127.0.0.1:3001",Hosts:[]string{"b.example.com"},ListenPort:8080,ListenIPv4:true,Enabled:true});if err!=nil{t.Fatal(err)}
	req:=AccessBlacklistRequest{Host:"a.example.com",Port:8080,IP:"192.0.2.1",Action:"inspect"}
	res,err:=svc.AccessBlacklist(ctx,req);if err!=nil||res.RuleID!=a.ID||res.Exact{t.Fatalf("inspect: %+v %v",res,err)}
	req.Action="block";req.RuleID=b.ID
	if _,err:=svc.AccessBlacklist(ctx,req);err==nil{t.Fatal("stale rule ID accepted")}
	req.RuleID=a.ID
	res,err=svc.AccessBlacklist(ctx,req);if err!=nil||!res.Changed||!res.Exact{t.Fatalf("block: %+v %v",res,err)}
	res,err=svc.AccessBlacklist(ctx,req);if err!=nil||res.Changed{t.Fatalf("duplicate: %+v %v",res,err)}
	req.Action="unblock"
	res,err=svc.AccessBlacklist(ctx,req);if err!=nil||!res.Changed||res.Exact{t.Fatalf("unblock: %+v %v",res,err)}
	a,err=store.Get(ctx,a.ID);if err!=nil||len(a.Security.IPBlacklist)!=1||a.Security.IPBlacklist[0]!="198.51.100.0/24"{t.Fatalf("existing blacklist changed: %v %v",a.Security.IPBlacklist,err)}
	b,err=store.Get(ctx,b.ID);if err!=nil||len(b.Security.IPBlacklist)!=0{t.Fatal("other rule changed")}
}
