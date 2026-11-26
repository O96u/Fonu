package api

import (
 "context"
 "fmt"
 "io"
 "net"
 "net/http"
 "net/netip"
 "net/url"
 "strconv"
 "strings"
 "sync"
 "time"
)

const faviconMax = 256 * 1024

type faviconEntry struct {
 data []byte
 mime string
 expires time.Time
}
var faviconCache = struct {
 sync.Mutex
 entries map[string]faviconEntry
 inflight map[string]chan struct{}
}{entries: make(map[string]faviconEntry), inflight: make(map[string]chan struct{})}
var faviconSlots = make(chan struct{}, 4)

// Only public addresses are allowed, including after DNS resolution.
// Pin the actual dial to a checked address to prevent DNS rebinding.
func faviconPublic(ip netip.Addr) bool {
 ip = ip.Unmap()
 if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() { return false }
 for _, prefix := range []string{"100.64.0.0/10","192.0.0.0/24","192.0.2.0/24","198.18.0.0/15","198.51.100.0/24","203.0.113.0/24","240.0.0.0/4","2001:db8::/32","64:ff9b::/96","64:ff9b:1::/48","2002::/16","2001::/23"} {
  if netip.MustParsePrefix(prefix).Contains(ip) { return false }
 }
 return true
}
func faviconDial(ctx context.Context, network, addr string) (net.Conn,error) {
 host, port, err := net.SplitHostPort(addr)
 if err != nil { return nil,err }
 ips, err := net.DefaultResolver.LookupNetIP(ctx,"ip",host)
 if err != nil { return nil,err }
 for _, ip := range ips { if !faviconPublic(ip) { return nil,fmt.Errorf("non-public favicon address") } }
 for _, ip := range ips {
  conn, err := (&net.Dialer{Timeout:2*time.Second}).DialContext(ctx,network,net.JoinHostPort(ip.String(),port))
  if err == nil { return conn,nil }
 }
 return nil,fmt.Errorf("favicon unavailable")
}
func faviconURL(domain string, port int, https bool) (string,bool) {
 domain = strings.TrimSpace(domain)
 if domain == "" || strings.ContainsAny(domain,"/*:@?#\\[] ") || net.ParseIP(domain)!=nil || !strings.Contains(domain,".") || port<1 || port>65535 { return "",false }
 scheme := "http"
 if https { scheme = "https" }
 host := domain
 if (https && port!=443) || (!https && port!=80) { host=net.JoinHostPort(domain,strconv.Itoa(port)) }
 return (&url.URL{Scheme:scheme,Host:host,Path:"/favicon.ico"}).String(),true
}
func fetchFavicon(ctx context.Context, target string) faviconEntry {
 result := faviconEntry{expires:time.Now().Add(15*time.Minute)}
 transport := &http.Transport{DialContext:faviconDial,DisableKeepAlives:true, TLSHandshakeTimeout:2*time.Second}
 defer transport.CloseIdleConnections()
 client := &http.Client{Transport:transport,Timeout:4*time.Second,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 req,err:=http.NewRequestWithContext(ctx,http.MethodGet,target,nil)
 if err!=nil{return result}
 req.Header.Set("Accept","image/png,image/x-icon,image/vnd.microsoft.icon,image/jpeg,image/gif,image/webp")
 resp,err:=client.Do(req)
 if err!=nil{return result}
 defer resp.Body.Close()
 if resp.StatusCode!=http.StatusOK{return result}
 data,err:=io.ReadAll(io.LimitReader(resp.Body,faviconMax+1))
 if err!=nil || len(data)==0 || len(data)>faviconMax{return result}
 mime:=http.DetectContentType(data)
 switch mime {
 case "image/png","image/jpeg","image/gif","image/webp","image/x-icon","image/vnd.microsoft.icon":
 default:return result
 }
 result.data,result.mime,result.expires=data,mime,time.Now().Add(24*time.Hour)
 return result
}
func (h *ProxyHandler) Favicon(w http.ResponseWriter,r *http.Request) {
 id,err:=parseID(r.PathValue("id"))
 if err!=nil {http.NotFound(w,r);return}
 rule,err:=h.svc.Get(r.Context(),id)
 if err!=nil || !rule.Enabled {http.NotFound(w,r);return}
 domain,port:=rule.Domain,rule.ListenPort
 if len(rule.Hosts)>0 {domain=rule.Hosts[0].Hostname;if rule.Hosts[0].ListenPort!=nil {port=*rule.Hosts[0].ListenPort}}
 target,ok:=faviconURL(domain,port,rule.HTTPSEnabled)
 if !ok {http.NotFound(w,r);return}
 entry,ok:=cachedFavicon(r.Context(),target)
 if !ok {w.Header().Set("Retry-After","2");w.WriteHeader(http.StatusServiceUnavailable);return}
 if len(entry.data)==0 {w.Header().Set("Cache-Control","private, max-age=900");http.NotFound(w,r);return}
 w.Header().Set("Content-Type",entry.mime)
 w.Header().Set("X-Content-Type-Options","nosniff")
 w.Header().Set("Cache-Control","private, max-age=86400")
 _,_=w.Write(entry.data)
}

// Concurrent requests for the same URL share a bounded fetch. A canceled
// browser request does not cancel the other waiters' shared result.
func cachedFavicon(ctx context.Context,target string)(faviconEntry,bool){
 for {
  faviconCache.Lock()
  if entry,ok:=faviconCache.entries[target];ok && time.Now().Before(entry.expires){
   faviconCache.Unlock();return entry,true
  }
  if done,ok:=faviconCache.inflight[target];ok{
   faviconCache.Unlock()
   select{case <-done:continue;case <-ctx.Done():return faviconEntry{},false}
  }
  select{
  case faviconSlots<-struct{}{}:
  default:faviconCache.Unlock();return faviconEntry{},false
  }
  done:=make(chan struct{})
  faviconCache.inflight[target]=done
  faviconCache.Unlock()
  fetchCtx,cancel:=context.WithTimeout(context.Background(),4*time.Second)
  entry:=fetchFavicon(fetchCtx,target)
  cancel()
  faviconCache.Lock()
  if len(faviconCache.entries)>=128 {
   for k,v:=range faviconCache.entries{if time.Now().After(v.expires){delete(faviconCache.entries,k)}}
   if len(faviconCache.entries)>=128{for k:=range faviconCache.entries{delete(faviconCache.entries,k);break}}
  }
  faviconCache.entries[target]=entry
  delete(faviconCache.inflight,target)
  close(done)
  <-faviconSlots
  faviconCache.Unlock()
  return entry,true
 }
}
