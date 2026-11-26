package service
import("context";"net/http";"net/http/httptest";"testing";"path/filepath";"github.com/fonu/fonu/internal/db";"github.com/fonu/fonu/internal/proxy")
func TestProbeProxyNoRedirect(t *testing.T){
 called:=false
 dest:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){called=true}));defer dest.Close()
 origin:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.Method!="HEAD"{t.Error("expected HEAD")};http.Redirect(w,r,dest.URL,302)}));defer origin.Close()
 d:=probeProxy(context.Background(),origin.URL)
 if d.State!="reachable"||d.HTTPStatus!=302||called||d.Metrics.TCPMS==nil||d.Metrics.TTFBMS==nil||d.Metrics.RemoteAddress==""||d.Metrics.Protocol==""{t.Fatalf("%+v redirect=%v",d,called)}
 tls:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){}));defer tls.Close()
 if d:=probeProxy(context.Background(),tls.URL);d.State!="tls_error"{t.Fatalf("%+v",d)}
}
func TestDiagnosticHistoryBounded(t *testing.T){
 ctx:=context.Background()
 conn,err:=db.Open(filepath.Join(t.TempDir(),"test.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 if err=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 target:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.WriteHeader(503)}));defer target.Close()
 store:=proxy.NewStore(conn)
 rule,err:=store.Create(ctx,proxy.CreateInput{Upstream:target.URL,Hosts:[]string{"diagnostic.example.com"},ListenPort:8080,ListenIPv4:true,Enabled:true});if err!=nil{t.Fatal(err)}
 s:=&ProxyService{db:conn,store:store}
 for i:=0;i<7;i++{d,err:=s.RunProxyDiagnostic(ctx,rule.ID);if err!=nil||d.State!="http_error"||d.HTTPStatus!=503{t.Fatalf("%+v %v",d,err)}}
 rows,err:=s.ProxyDiagnostics(ctx,rule.ID);if err!=nil||len(rows)!=5||rows[0].ID<=rows[4].ID||rows[0].Metrics.TTFBMS==nil{t.Fatalf("%+v %v",rows,err)}
 if err=store.Delete(ctx,rule.ID);err!=nil{t.Fatal(err)}
 var n int;if err=conn.QueryRowContext(ctx,"SELECT count(*) FROM proxy_diagnostics").Scan(&n);err!=nil||n!=0{t.Fatalf("orphan history %d %v",n,err)}
}
