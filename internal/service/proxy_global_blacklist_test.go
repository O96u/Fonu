package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/settings"
)

func TestGlobalBlacklistRequiresConfirmationAndPreservesPolicy(t *testing.T){
	ctx:=context.Background()
	dir:=t.TempDir()
	conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
	defer conn.Close()
	if err=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
	cfg:=config.Config{DataDir:dir,NginxBin:"fonu-missing-nginx-binary",NginxPIDFile:filepath.Join(dir,"nginx.pid"),NginxMimeTypes:filepath.Join(dir,"mime.types")}
	st:=settings.NewStore(conn)
	store:=proxy.NewStore(conn)
	svc:=NewProxyService(cfg,conn,store,certificate.NewStore(conn),st,nginx.NewManager(cfg,slog.Default()))
	if err=svc.SaveSettingsWithPublication(ctx,map[string]string{settings.KeyGlobalIPBlacklist:`["198.51.100.0/24"]`,settings.KeyGlobalIPWhitelist:`["192.0.2.50"]`});err!=nil{t.Fatal(err)}
	req:=GlobalBlacklistRequest{IP:"192.0.2.1",Action:"inspect"}
	result,err:=svc.GlobalBlacklist(ctx,req);if err!=nil||result.Exact||result.Changed{t.Fatalf("inspect: %+v %v",result,err)}
	req.Action="block"
	if _,err=svc.GlobalBlacklist(ctx,req);err==nil{t.Fatal("unconfirmed block accepted")}
	raw,_:=st.Get(ctx,settings.KeyGlobalIPBlacklist);if raw!=`["198.51.100.0/24"]`{t.Fatal("unconfirmed request mutated policy")}
	req.Confirmed=true
	result,err=svc.GlobalBlacklist(ctx,req);if err!=nil||!result.Exact||!result.Changed{t.Fatalf("block: %+v %v",result,err)}
	result,err=svc.GlobalBlacklist(ctx,req);if err!=nil||result.Changed{t.Fatalf("duplicate: %+v %v",result,err)}
	raw,_=st.Get(ctx,settings.KeyGlobalIPBlacklist)
	var list []string;if err=json.Unmarshal([]byte(raw),&list);err!=nil||len(list)!=2||list[1]!="192.0.2.1"{t.Fatalf("persisted: %s %v",raw,err)}
	req.IP="198.51.100.22"
	result,err=svc.GlobalBlacklist(ctx,req);if err!=nil||!result.Covered||result.Changed{t.Fatalf("CIDR coverage: %+v %v",result,err)}
	req.Action="unblock"
	result,err=svc.GlobalBlacklist(ctx,req);if err!=nil||result.Changed{t.Fatal("CIDR removed")}
	req.IP="192.0.2.1";req.Confirmed=false
	if _,err=svc.GlobalBlacklist(ctx,req);err==nil{t.Fatal("unconfirmed unblock accepted")}
	req.Confirmed=true
	result,err=svc.GlobalBlacklist(ctx,req);if err!=nil||result.Exact||!result.Changed{t.Fatalf("unblock: %+v %v",result,err)}
	raw,_=st.Get(ctx,settings.KeyGlobalIPBlacklist);if raw!=`["198.51.100.0/24"]`{t.Fatalf("existing CIDR changed: %s",raw)}
	raw,_=st.Get(ctx,settings.KeyGlobalIPWhitelist);if raw!=`["192.0.2.50"]`{t.Fatal("whitelist changed")}
}
