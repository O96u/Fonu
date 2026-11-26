package traffic

import (
	"testing"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/logstore"
)
func TestHighRateSamplesPreserveBytes(t *testing.T){
	c:=NewCollector(nil,"",nil)
	for i:=0;i<1000;i++{c.ingestEntry(logstore.AccessEntry{Domain:"test.example.com",ServerPort:8080,RequestLength:100,BytesSent:200},true)}
	r:=proxy.Rule{ID:1,ListenPort:8080,Hosts:[]proxy.Host{{Hostname:"test.example.com"}}}
	s:=c.SnapshotForRules([]proxy.Rule{r})[0]
	if s.UploadTotal!=100000||s.DownloadTotal!=200000||s.UploadRate!=20000||s.DownloadRate!=40000{t.Fatalf("unexpected traffic: %+v",s)}
}
