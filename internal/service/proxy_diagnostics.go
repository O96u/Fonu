package service

import (
 "context"
 "crypto/x509"
 "crypto/tls"
 "encoding/json"
 "net/http/httptrace"
 "sync"
 "errors"
 "fmt"
 "net"
 "net/http"
 "net/url"
 "time"

 "github.com/fonu/fonu/internal/validate"
)

type ProxyDiagnostic struct {
 ID int64 `json:"id"`
 Upstream string `json:"upstream"`
 CheckedAt string `json:"checked_at"`
 State string `json:"state"`
 DurationMS int64 `json:"duration_ms"`
 HTTPStatus int `json:"http_status"`
 Detail string `json:"detail"`
 Metrics DiagnosticMetrics `json:"metrics"`
}

type DiagnosticMetrics struct{
 DNSMS *int64 `json:"dns_ms,omitempty"`
 TCPMS *int64 `json:"tcp_ms,omitempty"`
 TLSMS *int64 `json:"tls_ms,omitempty"`
 TTFBMS *int64 `json:"ttfb_ms,omitempty"`
 RemoteAddress string `json:"remote_address,omitempty"`
 Protocol string `json:"protocol,omitempty"`
 TLSVersion string `json:"tls_version,omitempty"`
 CertificateExpires string `json:"certificate_expires,omitempty"`
}
func elapsedMS(t time.Time)*int64{if t.IsZero(){return nil};n:=time.Since(t).Milliseconds();return &n}

// Only the saved upstream is probed. No arbitrary request URL, redirects,
// proxy environment, authorization headers or response body downloads.
func probeProxy(ctx context.Context, upstream string) ProxyDiagnostic {
 start:=time.Now()
 out:=ProxyDiagnostic{Upstream:upstream,CheckedAt:start.UTC().Format(time.RFC3339Nano),State:"failed"}
 var mu sync.Mutex
 metrics:=DiagnosticMetrics{}
 finish:=func(state,detail string)ProxyDiagnostic{mu.Lock();out.Metrics=metrics;mu.Unlock();out.State=state;out.Detail=detail;out.DurationMS=time.Since(start).Milliseconds();return out}
 target,err:=validate.Upstream(upstream);if err!=nil{return finish("invalid","目标地址无效")}
 ctx,cancel:=context.WithTimeout(ctx,5*time.Second);defer cancel()
 tr:=&http.Transport{DialContext:(&net.Dialer{Timeout:3*time.Second}).DialContext,TLSHandshakeTimeout:3*time.Second,ResponseHeaderTimeout:4*time.Second,DisableKeepAlives:true,MaxResponseHeaderBytes:64<<10}
 defer tr.CloseIdleConnections()
 client:=&http.Client{Transport:tr,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 req,err:=http.NewRequestWithContext(ctx,http.MethodHead,target,nil);if err!=nil{return finish("invalid","目标地址无效")}
 var dnsStart,tlsStart time.Time
 attempts:=map[string]time.Time{}
 trace:=&httptrace.ClientTrace{
 DNSStart:func(httptrace.DNSStartInfo){mu.Lock();dnsStart=time.Now();mu.Unlock()},
 DNSDone:func(httptrace.DNSDoneInfo){mu.Lock();metrics.DNSMS=elapsedMS(dnsStart);mu.Unlock()},
 ConnectStart:func(network,addr string){mu.Lock();attempts[network+" "+addr]=time.Now();mu.Unlock()},
 ConnectDone:func(network,addr string,err error){mu.Lock();if err==nil{metrics.TCPMS=elapsedMS(attempts[network+" "+addr])};delete(attempts,network+" "+addr);mu.Unlock()},
 TLSHandshakeStart:func(){mu.Lock();tlsStart=time.Now();mu.Unlock()},
 TLSHandshakeDone:func(state tls.ConnectionState,err error){mu.Lock();defer mu.Unlock();metrics.TLSMS=elapsedMS(tlsStart);if state.Version!=0{metrics.TLSVersion=tls.VersionName(state.Version)};if len(state.PeerCertificates)>0{metrics.CertificateExpires=state.PeerCertificates[0].NotAfter.UTC().Format(time.RFC3339)}},
 GotConn:func(info httptrace.GotConnInfo){mu.Lock();defer mu.Unlock();if info.Conn!=nil{metrics.RemoteAddress=info.Conn.RemoteAddr().String()}},
 GotFirstResponseByte:func(){mu.Lock();metrics.TTFBMS=elapsedMS(start);mu.Unlock()},
 }
 req=req.WithContext(httptrace.WithClientTrace(req.Context(),trace))
 req.Header.Set("User-Agent","Fonu-Diagnostics")
 res,err:=client.Do(req)
 if err!=nil{
  var unknown x509.UnknownAuthorityError
  var invalid x509.CertificateInvalidError
  var hostname x509.HostnameError
  var dns *net.DNSError
  var netErr net.Error
  switch{
  case errors.As(err,&unknown)||errors.As(err,&invalid)||errors.As(err,&hostname):return finish("tls_error","上游证书无法通过系统信任校验（不代表 Nginx 配置失败）")
  case errors.Is(err,context.DeadlineExceeded)||(errors.As(err,&netErr)&&netErr.Timeout()):return finish("timeout","连接或响应超时")
  case errors.As(err,&dns):return finish("dns_error","目标域名解析失败")
  default:
   var ue *url.Error
   if errors.As(err,&ue){return finish("failed","无法完成 HTTP 请求：检查上游服务、端口及 TLS")}
   return finish("failed","无法完成 HTTP 请求")
  }
 }
 defer res.Body.Close()
 out.HTTPStatus=res.StatusCode
 mu.Lock();metrics.Protocol=res.Proto;mu.Unlock()
 if res.StatusCode>=500{return finish("http_error","上游已响应，但返回 5xx")}
 return finish("reachable","上游已响应 HTTP；不代表前端域名、登录或应用功能正常")
}

func(s *ProxyService) ProxyDiagnostics(ctx context.Context,id int64)([]ProxyDiagnostic,error){
 if _,err:=s.Get(ctx,id);err!=nil{return nil,err}
 rows,err:=s.db.QueryContext(ctx,"SELECT id,upstream,checked_at,state,duration_ms,http_status,detail,metrics_json FROM proxy_diagnostics WHERE rule_id=? ORDER BY id DESC LIMIT 5",id)
 if err!=nil{return nil,err};defer rows.Close()
 out:=[]ProxyDiagnostic{}
 for rows.Next(){var d ProxyDiagnostic;var raw string;if err:=rows.Scan(&d.ID,&d.Upstream,&d.CheckedAt,&d.State,&d.DurationMS,&d.HTTPStatus,&d.Detail,&raw);err!=nil{return nil,err};_ = json.Unmarshal([]byte(raw),&d.Metrics);out=append(out,d)}
 return out,rows.Err()
}
func(s *ProxyService) RunProxyDiagnostic(ctx context.Context,id int64)(ProxyDiagnostic,error){
 r,err:=s.Get(ctx,id);if err!=nil{return ProxyDiagnostic{},err}
 d:=probeProxy(ctx,r.Upstream)
 if ctx.Err()!=nil{return d,ctx.Err()}
 tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return d,err};defer tx.Rollback()
 raw,err:=json.Marshal(d.Metrics);if err!=nil{return d,err}
 result,err:=tx.ExecContext(ctx,"INSERT INTO proxy_diagnostics(rule_id,upstream,checked_at,state,duration_ms,http_status,detail,metrics_json) SELECT id,?,?,?,?,?,?,? FROM proxy_rules WHERE id=? AND upstream=?",d.Upstream,d.CheckedAt,d.State,d.DurationMS,d.HTTPStatus,d.Detail,string(raw),id,r.Upstream)
 if err!=nil{return d,err}
 if n,err:=result.RowsAffected();err!=nil||n!=1{return d,fmt.Errorf("规则已变化，请重新检测")}
 d.ID,err=result.LastInsertId();if err!=nil{return d,err}
 if _,err=tx.ExecContext(ctx,"DELETE FROM proxy_diagnostics WHERE rule_id=? AND id NOT IN (SELECT id FROM proxy_diagnostics WHERE rule_id=? ORDER BY id DESC LIMIT 5)",id,id);err!=nil{return d,err}
 return d,tx.Commit()
}
