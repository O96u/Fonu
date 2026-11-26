package nginx
import("context";"fmt";"io";"log/slog";"net";"net/http";"net/http/httptest";"os";"os/exec";"path/filepath";"testing";"time";"github.com/fonu/fonu/internal/config";"github.com/fonu/fonu/internal/proxy")
func TestTrustedProxyRealNginx(t *testing.T){
 if _,err:=exec.LookPath("nginx");err!=nil{t.Skip("nginx not installed")}
 dir:=t.TempDir();mime:=filepath.Join(dir,"mime.types");if err:=os.WriteFile(mime,[]byte("types { text/html html; }\n"),0644);err!=nil{t.Fatal(err)}
 cfg:=config.Config{DataDir:dir,NginxBin:"nginx",NginxPIDFile:filepath.Join(dir,"nginx","nginx.pid"),NginxMimeTypes:mime}
 mgr:=NewManager(cfg,slog.Default());ctx,cancel:=context.WithTimeout(context.Background(),45*time.Second);defer cancel();defer mgr.Stop(context.Background())
 upstream:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,r.Header.Get("X-Real-IP"))}));defer upstream.Close()
 listener,err:=net.Listen("tcp4","127.0.0.1:0");if err!=nil{t.Fatal(err)};port:=listener.Addr().(*net.TCPAddr).Port;listener.Close()
 rules:=[]proxy.Rule{{ID:1,Upstream:upstream.URL,ListenPort:port,ListenIPv4:true,Hosts:[]proxy.Host{{Hostname:"nas.example.com"}},Enabled:true}}
 check:=func(want string){t.Helper();client:=http.Client{Timeout:3*time.Second,Transport:&http.Transport{DisableKeepAlives:true}};req,err:=http.NewRequestWithContext(ctx,http.MethodGet,fmt.Sprintf("http://127.0.0.1:%d/",port),nil);if err!=nil{t.Fatal(err)};req.Host="nas.example.com";req.Header.Set("X-Forwarded-For","8.8.8.8");req.Header.Set("CF-Connecting-IP","8.8.4.4");res,err:=client.Do(req);if err!=nil{t.Fatal(err)};defer res.Body.Close();body,_:=io.ReadAll(res.Body);if res.StatusCode!=200||string(body)!=want{t.Fatalf("want %s got %d %s",want,res.StatusCode,body)}}
 for _,v:=range []struct{cfg TrustedProxyConfig;want string}{{TrustedProxyConfig{},"127.0.0.1"},{TrustedProxyConfig{Enabled:true,Preset:"cloudflare"},"127.0.0.1"},{TrustedProxyConfig{Enabled:true,Preset:"custom",CIDRs:[]string{"127.0.0.1"},Header:"X-Forwarded-For"},"8.8.8.8"}}{
  if _,err:=mgr.ApplyWithOptions(ctx,rules,nil,GenerateOptions{TrustedProxy:v.cfg});err!=nil{t.Fatal(err)};check(v.want)
 }
}
