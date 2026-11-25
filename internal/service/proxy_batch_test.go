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

func TestSaveBatchAtomic(t *testing.T){
	ctx:=context.Background()
	dir:=t.TempDir()
	conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
	defer conn.Close()
	if err=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
	cfg:=config.Config{DataDir:dir,NginxBin:"fonu-missing-nginx-binary",NginxPIDFile:filepath.Join(dir,"nginx.pid"),NginxMimeTypes:filepath.Join(dir,"mime.types")}
	store:=proxy.NewStore(conn)
	svc:=NewProxyService(cfg,conn,store,certificate.NewStore(conn),nil,nginx.NewManager(cfg,slog.Default()))
	first:=proxy.CreateInput{Upstream:"http://127.0.0.1:3000",Hosts:[]string{"a.example.com"},ListenPort:8080,ListenIPv4:true,Enabled:true}
	second:=first
	second.Hosts=[]string{"b.example.com"}
	second.Upstream="invalid"
	if _,err:=svc.SaveBatch(ctx,0,nil,[]proxy.CreateInput{first,second});err==nil{t.Fatal("invalid batch accepted")}
	rows,err:=store.List(ctx);if err!=nil||len(rows)!=0{t.Fatalf("partial batch persisted: %v %d",err,len(rows))}
	second.Upstream=first.Upstream
	if rows,err=svc.SaveBatch(ctx,0,nil,[]proxy.CreateInput{first,second});err!=nil||len(rows)!=2{t.Fatalf("valid batch failed: %v",err)}
}
