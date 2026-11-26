package publicip

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var ipv4Providers = []string{
	"https://ipv4.icanhazip.com",
	"https://ifconfig.me/ip",
	"https://myip.ipip.net",
	"https://api.ipify.org",
}

var ipv6Providers = []string{
	"https://ipv6.icanhazip.com",
	"https://api6.ipify.org",
	"https://api64.ipify.org",
}

// Probe the two address families independently; IPv4 failures cannot consume
// the IPv6 budget, and dual-stack providers are dialled on the requested family.
func Detect(ctx context.Context) (ipv4, ipv6 string, err error) {
 ctx,cancel:=context.WithTimeout(ctx,6*time.Second);defer cancel()
 type result struct{ip string;err error}
 v4,v6:=make(chan result,1),make(chan result,1)
 go func(){ip,e:=detectIPv4(ctx);v4<-result{ip,e}}()
 go func(){ip,e:=detectIPv6(ctx);v6<-result{ip,e}}()
 a,b:=<-v4,<-v6
 if a.ip!=""||b.ip!=""{return a.ip,b.ip,nil}
 return "", "", fmt.Errorf("未检测到公网 IP：%w",a.err)
}

func detectClient(network string)*http.Client{
 dialer:=&net.Dialer{Timeout:2*time.Second,KeepAlive:-1}
 return &http.Client{Timeout:2500*time.Millisecond,
 Transport:&http.Transport{Proxy:nil,DisableKeepAlives:true,DialContext:func(ctx context.Context,_,addr string)(net.Conn,error){return dialer.DialContext(ctx,network,addr)}},
 CheckRedirect:func(req *http.Request,via []*http.Request)error{return http.ErrUseLastResponse},
 }
}

func detectFamily(ctx context.Context,providers []string,wantV6 bool)(string,error){
 network:="tcp4";if wantV6{network="tcp6"}
 client:=detectClient(network);defer client.CloseIdleConnections()
 var lastErr error
 for _,endpoint:=range providers{
  if err:=ctx.Err();err!=nil{return "",err}
  ip,err:=fetchIP(ctx,client,endpoint,wantV6)
  if err==nil{return ip,nil};lastErr=err
 }
 if lastErr==nil{lastErr=fmt.Errorf("没有可用探测服务")}
 return "",lastErr
}
func detectIPv4(ctx context.Context)(string,error){return detectFamily(ctx,ipv4Providers,false)}
func detectIPv6(ctx context.Context)(string,error){return detectFamily(ctx,ipv6Providers,true)}

func fetchIP(ctx context.Context, client *http.Client, url string, wantV6 bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("invalid ip response")
	}
	if wantV6 && parsed.To4() != nil {
		return "", fmt.Errorf("no ipv6")
	}
	if !wantV6 && parsed.To4() == nil {
		return "", fmt.Errorf("no ipv4")
	}
	if !wantV6 && !IsPublicIPv4(ip) {
		return "", fmt.Errorf("private ipv4")
	}
	if wantV6 && !IsPublicIPv6(ip) {
		return "", fmt.Errorf("private ipv6")
	}
	return ip, nil
}

func IsPublicIPv4(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil || parsed.To4() == nil {
		return false
	}
	return !parsed.IsPrivate() && !parsed.IsLoopback() && !parsed.IsLinkLocalUnicast() && !parsed.IsUnspecified()
}

func IsPublicIPv6(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil || parsed.To4() != nil {
		return false
	}
	return !parsed.IsPrivate() && !parsed.IsLoopback() && !parsed.IsLinkLocalUnicast() && !parsed.IsUnspecified()
}
