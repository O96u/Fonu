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
 "github.com/fonu/fonu/internal/settings"
)
func TestGlobalSettingsPublicationRollback(t *testing.T){
 ctx:=context.Background()
 dir:=t.TempDir()
 conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
 defer conn.Close()
 if err=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 cfg:=config.Config{DataDir:dir,NginxBin:"fonu-missing-nginx-binary",NginxPIDFile:filepath.Join(dir,"nginx.pid"),NginxMimeTypes:filepath.Join(dir,"mime.types")}
 st:=settings.NewStore(conn)
 svc:=NewProxyService(cfg,conn,proxy.NewStore(conn),certificate.NewStore(conn),st,nginx.NewManager(cfg,slog.Default()))
 if err=svc.SaveSettingsWithPublication(ctx,map[string]string{settings.KeyTrustedProxy:`{"enabled":true,"cidrs":["0.0.0.0/0"]}`});err==nil{t.Fatal("unsafe setting accepted")}
 raw,err:=st.Get(ctx,settings.KeyTrustedProxy);if err!=nil||raw!=""{t.Fatalf("setting persisted: %s %v",raw,err)}
 if err=svc.SaveSettingsWithPublication(ctx,map[string]string{settings.KeyTrustedProxy:`{"enabled":true,"cidrs":["10.0.0.1"]}`});err!=nil{t.Fatal(err)}
 raw,err=st.Get(ctx,settings.KeyTrustedProxy);if err!=nil||raw==""{t.Fatal("valid setting not committed")}
}
