package nginx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestApplyWithNginxIfAvailable(t *testing.T) {
	if _,err:=exec.LookPath("nginx");err!=nil{t.Skip("nginx not installed")}
	dir:=t.TempDir()
	mimePath:=filepath.Join(dir,"mime.types")
	if err:=os.WriteFile(mimePath,[]byte("types { text/html html; }\n"),0644);err!=nil{t.Fatal(err)}
	cfg:=config.Config{DataDir:dir,NginxBin:"nginx",NginxPIDFile:filepath.Join(dir,"nginx","nginx.pid"),NginxMimeTypes:mimePath}
	mgr:=NewManager(cfg,slog.New(slog.NewTextHandler(os.Stdout,nil)))
	ctx,cancel:=context.WithTimeout(context.Background(),45*time.Second)
	defer cancel()
	defer func(){if err:=mgr.Stop(context.Background());err!=nil{t.Error(err)}}()
	backend:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,"first")}))
	defer backend.Close()
	second:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,"second")}))
	defer second.Close()
	listener,err:=net.Listen("tcp4","127.0.0.1:0");if err!=nil{t.Fatal(err)}
	port:=listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	rules:=[]proxy.Rule{{ID:1,Upstream:backend.URL,ListenPort:port,ListenIPv4:true,Hosts:[]proxy.Host{{Hostname:"nas.example.com"}},Enabled:true}}
	check:=func(want string){
		t.Helper()
		client:=http.Client{Timeout:2*time.Second,Transport:&http.Transport{DisableKeepAlives:true}}
		req,err:=http.NewRequestWithContext(ctx,http.MethodGet,fmt.Sprintf("http://127.0.0.1:%d/",port),nil)
		if err!=nil{t.Fatal(err)}
		req.Host="nas.example.com"
		res,err:=client.Do(req);if err!=nil{t.Fatal(err)}
		defer res.Body.Close()
		body,err:=io.ReadAll(res.Body);if err!=nil||res.StatusCode!=200||string(body)!=want{t.Fatalf("response %d %q: %v",res.StatusCode,body,err)}
	}
	first,err:=mgr.Apply(ctx,rules,nil);if err!=nil{t.Fatal(err)}
	if workerConfirmationSupported()&&!first.Confirmed{t.Fatal("worker not confirmed")}
	check("first")
	same,err:=mgr.Apply(ctx,rules,nil);if err!=nil{t.Fatal(err)}
	if workerConfirmationSupported()&&!same.Unchanged{t.Fatal("unchanged publication should not reload")}
	rules[0].Upstream=second.URL
	changed,err:=mgr.Apply(ctx,rules,nil);if err!=nil{t.Fatal(err)}
	if changed.Unchanged||workerConfirmationSupported()&&!changed.Confirmed{t.Fatal("changed publication not confirmed")}
	check("second")
	if _,err=mgr.ApplyWithOptions(ctx,rules,nil,GenerateOptions{GlobalCustomMode:"custom",GlobalCustomOverrideSet:true,GlobalCustomOverride:"not_a_valid_nginx_directive on;"});err==nil{t.Fatal("invalid config accepted")}
	check("second")
}
