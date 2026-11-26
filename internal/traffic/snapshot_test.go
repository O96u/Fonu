package traffic
import("testing";"github.com/fonu/fonu/internal/proxy";"github.com/fonu/fonu/internal/logstore")
func TestSnapshotGloballyDeduplicatesEndpointsAndClients(t *testing.T){
 c:=NewCollector(nil,"",nil)
 for _,domain:=range []string{"a.example.com","b.example.com"}{c.ingestEntry(logstore.AccessEntry{Domain:domain,ServerPort:443,ClientIP:"192.0.2.1",RequestLength:100,BytesSent:200},true)}
 rules:=[]proxy.Rule{{ID:1,ListenPort:443,Hosts:[]proxy.Host{{Hostname:"a.example.com"}}},{ID:2,ListenPort:443,Hosts:[]proxy.Host{{Hostname:"b.example.com"}}},{ID:3,ListenPort:443,Hosts:[]proxy.Host{{Hostname:"a.example.com"}}}}
 s:=c.SnapshotAll(rules)
 if len(s.Rules)!=3||s.Total.Connections!=1||s.Total.UploadTotal!=200||s.Total.DownloadTotal!=400||s.ActiveWindowSeconds!=65||s.RateWindowSeconds!=5{t.Fatalf("%+v",s)}
 for _,r:=range s.Rules{if r.Connections!=1||r.UploadTotal!=100{t.Fatalf("%+v",r)}}
}
