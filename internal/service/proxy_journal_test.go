package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/db"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/settings"
)

func TestPendingPublicationRecovery(t *testing.T){
	for _,commit:=range []bool{false,true}{
		t.Run(map[bool]string{false:"uncommitted",true:"committed"}[commit],func(t *testing.T){
			ctx:=context.Background()
			dir:=t.TempDir()
			conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
			defer conn.Close()
			if err=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
			cfg:=config.Config{DataDir:dir,NginxBin:"fonu-missing-nginx-binary",NginxPIDFile:filepath.Join(dir,"nginx.pid"),NginxMimeTypes:filepath.Join(dir,"mime.types")}
			svc:=NewProxyService(cfg,conn,proxy.NewStore(conn),certificate.NewStore(conn),settings.NewStore(conn),nginx.NewManager(cfg,slog.Default()))
			if err=nginx.WriteGlobalCustom(cfg,"gzip off;");err!=nil{t.Fatal(err)}
			j,err:=svc.beginProxyJournal();if err!=nil{t.Fatal(err)}
			if err=nginx.WriteGlobalCustom(cfg,"gzip on;");err!=nil{t.Fatal(err)}
			if commit{
				if _,err=conn.ExecContext(ctx,"INSERT INTO proxy_publication_commit(id,token) VALUES(1,?)",j.Token);err!=nil{t.Fatal(err)}
			}
			if err=svc.ReloadAll(ctx);err!=nil{t.Fatal(err)}
			content,err:=nginx.ReadGlobalCustom(cfg);if err!=nil{t.Fatal(err)}
			want:="gzip off;";if commit{want="gzip on;"}
			if content!=want{t.Fatalf("got %q want %q",content,want)}
			if _,err=os.Stat(svc.journalPath());!os.IsNotExist(err){t.Fatal("journal not cleared")}
		})
	}
}


func TestPendingCIDRPublicationRecovery(t *testing.T){
	for _,test:=range []struct{name string;oldExists,committed bool}{
		{"restore old",true,false},{"remove uncommitted new",false,false},{"keep committed",true,true},
	}{
		t.Run(test.name,func(t *testing.T){
			ctx:=context.Background()
			dir:=t.TempDir()
			conn,err:=db.Open(filepath.Join(dir,"fonu.db"));if err!=nil{t.Fatal(err)}
			defer conn.Close()
			if err=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
			cfg:=config.Config{DataDir:dir,NginxBin:"fonu-missing-nginx-binary",NginxPIDFile:filepath.Join(dir,"nginx.pid"),NginxMimeTypes:filepath.Join(dir,"mime.types")}
			svc:=NewProxyService(cfg,conn,proxy.NewStore(conn),certificate.NewStore(conn),settings.NewStore(conn),nginx.NewManager(cfg,slog.Default()))
			if err=os.MkdirAll(cfg.NginxDir(),0755);err!=nil{t.Fatal(err)}
			old,newBody:=[]byte("10.0.0.0/8 1;\n"),[]byte("192.168.0.0/16 1;\n")
			if test.oldExists{if err=os.WriteFile(cfg.ChinaCIDRPath(),old,0644);err!=nil{t.Fatal(err)}}
			j,err:=svc.beginProxyJournalWithCIDR(&cidrSnapshot{Exists:test.oldExists,Body:old});if err!=nil{t.Fatal(err)}
			if err=os.WriteFile(cfg.ChinaCIDRPath(),newBody,0644);err!=nil{t.Fatal(err)}
			if test.committed{
				if _,err=conn.ExecContext(ctx,"INSERT INTO proxy_publication_commit(id,token) VALUES(1,?)",j.Token);err!=nil{t.Fatal(err)}
			}
			if err=svc.ReloadAll(ctx);err!=nil{t.Fatal(err)}
			body,err:=os.ReadFile(cfg.ChinaCIDRPath())
			if !test.oldExists&&!test.committed{if !os.IsNotExist(err){t.Fatalf("uncommitted new CIDR not removed: %v",err)}}else{
				want:=old;if test.committed{want=newBody}
				if err!=nil||string(body)!=string(want){t.Fatalf("got %q want %q: %v",body,want,err)}
			}
			if _,err=os.Stat(svc.journalPath());!os.IsNotExist(err){t.Fatal("journal not cleared")}
		})
	}
}
