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
