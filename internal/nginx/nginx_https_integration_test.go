package nginx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

// The frontend certificates must be selected by SNI on the same listener,
// and an uncovered SAN must not replace a working HTTPS configuration.
func TestApplyHTTPSWithMultipleCertificates(t *testing.T) {
	if _,err:=exec.LookPath("nginx");err!=nil{t.Skip("nginx not installed")}
	dir:=t.TempDir()
	mime:=filepath.Join(dir,"mime.types")
	if err:=os.WriteFile(mime,[]byte("types { text/html html; }\n"),0644);err!=nil{t.Fatal(err)}
	cfg:=config.Config{DataDir:dir,NginxBin:"nginx",NginxPIDFile:filepath.Join(dir,"nginx","nginx.pid"),NginxMimeTypes:mime}
	mgr:=NewManager(cfg,slog.Default())
	ctx,cancel:=context.WithTimeout(context.Background(),45*time.Second)
	defer cancel()
	defer func(){if err:=mgr.Stop(context.Background());err!=nil{t.Error(err)}}()
	backend:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,"secure")}))
	defer backend.Close()
	listener,err:=net.Listen("tcp4","127.0.0.1:0")
	if err!=nil{t.Fatal(err)}
	port:=listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	hosts:=[]string{"nas.example.com","pve.example.com"}
	var certs []CertSource
	var bindings []proxy.Host
	for _,host:=range hosts{
		path:=filepath.Join(cfg.CertsDir(),host)
		writeTestCertificate(t,path,[]string{host},time.Now().Add(time.Hour))
		certs=append(certs,CertSource{Domains:[]string{host},CertPath:filepath.Join(path,"fullchain.pem"),KeyPath:filepath.Join(path,"privatekey.pem")})
		bindings=append(bindings,proxy.Host{Hostname:host})
	}
	rules:=[]proxy.Rule{{ID:1,Upstream:backend.URL,ListenPort:port,ListenIPv4:true,Hosts:bindings,HTTPSEnabled:true,Enabled:true}}
	if result,err:=mgr.Apply(ctx,rules,certs);err!=nil||workerConfirmationSupported()&&!result.Confirmed{t.Fatalf("HTTPS apply failed: %v",err)}
	check:=func(){
		t.Helper()
		for i,host:=range hosts{
			roots:=x509.NewCertPool()
			pem,err:=os.ReadFile(certs[i].CertPath)
			if err!=nil||!roots.AppendCertsFromPEM(pem){t.Fatalf("load test trust root: %v",err)}
			tr:=&http.Transport{DisableKeepAlives:true,TLSClientConfig:&tls.Config{RootCAs:roots,ServerName:host,MinVersion:tls.VersionTLS12}}
			client:=http.Client{Timeout:3*time.Second,Transport:tr}
			req,err:=http.NewRequestWithContext(ctx,http.MethodGet,fmt.Sprintf("https://127.0.0.1:%d/",port),nil)
			if err!=nil{t.Fatal(err)}
			req.Host=host
			res,err:=client.Do(req)
			if err!=nil{t.Fatalf("%s TLS: %v",host,err)}
			body,err:=io.ReadAll(res.Body)
			res.Body.Close()
			tr.CloseIdleConnections()
			if err!=nil||res.StatusCode!=200||string(body)!="secure"{t.Fatalf("%s: status=%d body=%q err=%v",host,res.StatusCode,body,err)}
		}
	}
	check()
	rules[0].Hosts=append(rules[0].Hosts,proxy.Host{Hostname:"uncovered.example.com"})
	if _,err:=mgr.Apply(ctx,rules,certs);err==nil{t.Fatal("uncovered HTTPS host accepted")}
	check()
}
