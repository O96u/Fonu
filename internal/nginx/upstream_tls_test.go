package nginx

import (
	"strings"
	"testing"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestUpstreamTLSVerificationCompatibility(t *testing.T){
	cfg:=config.Config{DataDir:t.TempDir()}
	rule:=proxy.Rule{ID:1,Enabled:true,ListenPort:8080,ListenIPv4:true,Upstream:"https://pve.example.com:8006",Hosts:[]proxy.Host{{Hostname:"app.example.com"}}}
	legacy,err:=GenerateRuleBlocks(cfg,rule,nil,GenerateOptions{})
	if err!=nil||!strings.Contains(legacy,"proxy_ssl_verify off;"){t.Fatalf("legacy compatibility: %v",err)}
	rule.Security.ProxySSLVerifyConfigured=true
	secure,err:=GenerateRuleBlocks(cfg,rule,nil,GenerateOptions{})
	if systemCABundle()==""{if err==nil{t.Fatal("missing CA must fail closed")};return}
	if err!=nil||!strings.Contains(secure,"proxy_ssl_verify on;")||!strings.Contains(secure,"proxy_ssl_trusted_certificate "){t.Fatalf("verification missing: %v",err)}
	rule.Security.ProxySSLVerifyOff=true
	skip,err:=GenerateRuleBlocks(cfg,rule,nil,GenerateOptions{})
	if err!=nil||!strings.Contains(skip,"proxy_ssl_verify off;"){t.Fatalf("explicit skip: %v",err)}
}
