package nginx
import("strings";"testing")
func TestTrustedProxyValidationAndPreset(t *testing.T){
 cfg:=TrustedProxyConfig{Enabled:true,Preset:"cloudflare",Header:"X-Forwarded-For",Recursive:true}
 if err:=cfg.Validate();err!=nil{t.Fatal(err)}
 var b strings.Builder;writeRealIPDirectives(&b,cfg)
 if !strings.Contains(b.String(),"real_ip_header CF-Connecting-IP;")||strings.Contains(b.String(),"real_ip_recursive on;"){t.Fatal("Cloudflare header not pinned")}
 for _,cidr:=range []string{"0.0.0.0/0","::/0","1.2.3.4/0","::1/0","invalid"}{if err:=(TrustedProxyConfig{Enabled:true,Preset:"custom",CIDRs:[]string{cidr}}).Validate();err==nil{t.Fatalf("accepted %s",cidr)}}
 if err:=(TrustedProxyConfig{Enabled:true,Preset:"custom"}).Validate();err==nil{t.Fatal("empty trusted proxy accepted")}
 if err:=(TrustedProxyConfig{Enabled:true,Preset:"custom",CIDRs:[]string{"192.168.8.10"},Header:"X-Forwarded-For"}).Validate();err!=nil{t.Fatal(err)}
}
