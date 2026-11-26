package nginx

import (
	"strings"
	"testing"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestCatchAllDeduplicatesOverlappingFamilies(t *testing.T){
	cfg:=config.Config{DataDir:t.TempDir()}
	rules:=[]proxy.Rule{
		{ID:1,ListenPort:8080,ListenIPv4:true,ListenIPv6:true,Enabled:true,Upstream:"http://127.0.0.1:3000",Hosts:[]proxy.Host{{Hostname:"a.example.com"}}},
		{ID:2,ListenPort:8080,ListenIPv4:true,Enabled:true,Upstream:"http://127.0.0.1:3001",Hosts:[]proxy.Host{{Hostname:"b.example.com"}}},
	}
	content,err:=Generate(cfg,rules,nil,GenerateOptions{})
	if err!=nil{t.Fatal(err)}
	if strings.Count(content,"listen 8080 default_server;")!=1{t.Fatal("duplicate IPv4 default server")}
	if strings.Count(content,"listen [::]:8080 default_server;")!=1{t.Fatal("missing IPv6 default server")}
}

func TestMixedListenProtocolConflict(t *testing.T){
	err:=validateListenProtocols([]listenEndpoint{{Port:8080,IPv4:true},{Port:8080,IPv4:true,SSL:true}})
	if err==nil{t.Fatal("mixed protocol accepted")}
}
