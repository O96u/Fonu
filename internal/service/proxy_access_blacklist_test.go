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
	"github.com/fonu/fonu/internal/settings"
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
	svc:=NewProxyService(cfg,conn,store,certificate.NewStore(conn),settings.NewStore(conn),nginx.NewManager(cfg,slog.Default()))
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
	logReq:=LogBlacklistRequest{Host:"a.example.com",Port:8080,IP:"192.0.2.1",Action:"inspect"}
	lr,err:=svc.LogBlacklist(ctx,logReq);if err!=nil||lr.Scope!="rule"||lr.RuleID!=a.ID{t.Fatalf("local inspect: %+v %v",lr,err)}
	logReq.Action="block";logReq.Scope="global";logReq.Confirmed=true
	if _,err=svc.LogBlacklist(ctx,logReq);err==nil{t.Fatal("wrong scope accepted")}
	logReq.Scope="rule";logReq.RuleID=a.ID;logReq.Confirmed=false
	if _,err=svc.LogBlacklist(ctx,logReq);err==nil{t.Fatal("unconfirmed local mutation accepted")}
	logReq.Confirmed=true
	lr,err=svc.LogBlacklist(ctx,logReq);if err!=nil||!lr.Changed||!lr.Exact{t.Fatalf("local block: %+v %v",lr,err)}
	logReq.Host="203.0.113.10:8080"
	if _,err=svc.LogBlacklist(ctx,logReq);err==nil{t.Fatal("stale local confirmation expanded to global")}
	logReq.Action="inspect";logReq.Scope="";logReq.RuleID=0
	lr,err=svc.LogBlacklist(ctx,logReq);if err!=nil||lr.Scope!="global"||lr.Exact{t.Fatalf("unmatched inspect: %+v %v",lr,err)}
	logReq.Action="block";logReq.Scope="global"
	lr,err=svc.LogBlacklist(ctx,logReq);if err!=nil||!lr.Changed||!lr.Exact{t.Fatalf("global block: %+v %v",lr,err)}
	logReq.Action="unblock"
	lr,err=svc.LogBlacklist(ctx,logReq);if err!=nil||!lr.Changed||lr.Exact{t.Fatalf("global unblock: %+v %v",lr,err)}
	a,err=store.Get(ctx,a.ID);if err!=nil||len(a.Security.IPBlacklist)!=2{t.Fatal("global unblock modified local list")}
}

func TestLogBlacklistScopeMatching(t *testing.T){
	a:=proxy.Rule{ID:1,Domain:"*.example.com",ListenPort:443,Enabled:true}
	b:=proxy.Rule{ID:2,Domain:"nas.example.com",ListenPort:443,Enabled:true}
	for _,tc:=range []struct{host string;port int;want int64;fail bool}{
		{"nas.example.com",443,2,false},
		{"deep.a.example.com",443,1,false},
		{"203.0.113.10:443",443,0,false},
		{"nas.example.com",0,0,true},
		{"other.example.net",443,0,false},
	}{
		r,found,err:=logBlacklistRule([]proxy.Rule{a,b},tc.host,tc.port)
		if (err!=nil)!=tc.fail||(!tc.fail&&(found!=(tc.want!=0)||r.ID!=tc.want)){t.Fatalf("%+v: %+v %t %v",tc,r,found,err)}
	}
	b.NginxMode="custom"
	if _,_,err:=logBlacklistRule([]proxy.Rule{a,b},"nas.example.com",443);err==nil{t.Fatal("custom rule must not fall back to global")}
	if _,_,err:=logBlacklistRule([]proxy.Rule{a,a},"nas.example.com",443);err==nil{t.Fatal("ambiguous rule must not fall back to global")}
}
