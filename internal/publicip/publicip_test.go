package publicip

import (
 "context"
 "net"
 "net/http"
 "net/http/httptest"
 "testing"
 "time"
)

func TestIsPublicIPv4(t *testing.T) {
	if IsPublicIPv4("192.168.1.1") {
		t.Fatal("private ip should be rejected")
	}
	if !IsPublicIPv4("8.8.8.8") {
		t.Fatal("public ip should be accepted")
	}
}

func TestIPv6ProbeUsesIPv6Transport(t *testing.T){
 v4:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_,_=w.Write([]byte("2606:4700::1111"))}))
 defer v4.Close()
 ctx,cancel:=context.WithTimeout(context.Background(),2*time.Second);defer cancel()
 if ip,err:=detectFamily(ctx,[]string{v4.URL},true);err==nil||ip!=""{t.Fatal("IPv4 transport accepted for IPv6 probe")}
 listener,err:=net.Listen("tcp6","[::1]:0");if err!=nil{t.Skip("IPv6 loopback unavailable")}
 server:=httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){_,_=w.Write([]byte("2606:4700::1111"))}))
 _=server.Listener.Close();server.Listener=listener;server.Start();defer server.Close()
 ctx2,cancel2:=context.WithTimeout(context.Background(),2*time.Second);defer cancel2()
 if ip,err:=detectFamily(ctx2,[]string{server.URL},true);err!=nil||ip!="2606:4700::1111"{t.Fatalf("IPv6 probe: %q %v",ip,err)}
}
