package api
import("net/netip";"testing")
func TestFaviconPublic(t *testing.T){
 for _,s:=range []string{"127.0.0.1","10.0.0.1","169.254.169.254","100.64.0.1","::1","::ffff:192.168.1.1","fc00::1","64:ff9b::a00:1"}{if faviconPublic(netip.MustParseAddr(s)){t.Errorf("allowed %s",s)}}
 if !faviconPublic(netip.MustParseAddr("8.8.8.8")){t.Error("public IP rejected")}
}
func TestFaviconURL(t *testing.T){
 if got,ok:=faviconURL("example.com",8443,true);!ok||got!="https://example.com:8443/favicon.ico"{t.Fatalf("%s %v",got,ok)}
 for _,s:=range []string{"*.example.com","127.0.0.1","foo/bar","localhost","user@example.com"}{if _,ok:=faviconURL(s,443,true);ok{t.Errorf("allowed %s",s)}}
}
