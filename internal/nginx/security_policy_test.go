package nginx

import (
	"strings"
	"testing"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)
func TestGlobalDenyBeforeAllow(t *testing.T){
	cfg:=config.Config{DataDir:t.TempDir()}
	r:=proxy.Rule{ID:1,Enabled:true,ListenPort:8080,ListenIPv4:true,Upstream:"http://127.0.0.1:3000",Hosts:[]proxy.Host{{Hostname:"app.example.com"}}}
	opts:=GenerateOptions{GlobalIPBlacklist:[]string{"10.1.2.3"},GlobalIPWhitelist:[]string{"10.1.2.3"}}
	out,err:=Generate(cfg,[]proxy.Rule{r},nil,opts)
	if err!=nil{t.Fatal(err)}
	for _,block:=range strings.Split(out,"location / {")[:2]{
		d,a:=strings.Index(block,"deny 10.1.2.3;"),strings.Index(block,"allow 10.1.2.3;")
		if d<0||a<0||d>=a{t.Fatal("global deny must win")}
	}
}
func TestUnsafeTrustedProxyRejected(t *testing.T){
	for _,c:=range []TrustedProxyConfig{
		{Enabled:true,CIDRs:[]string{"0.0.0.0/0"}},
		{Enabled:true,CIDRs:[]string{"10.0.0.1"},Header:"X-Real-IP;"},
		{Enabled:true,CIDRs:[]string{"10.0.0.1;"}},
	}{
		if _,err:=Generate(config.Config{DataDir:t.TempDir()},nil,nil,GenerateOptions{TrustedProxy:c});err==nil{t.Fatal("unsafe trusted proxy accepted")}
	}
}
