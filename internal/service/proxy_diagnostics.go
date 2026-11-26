package service

import (
 "context"
 "crypto/x509"
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
}

// Only the saved upstream is probed. No arbitrary request URL, redirects,
// proxy environment, authorization headers or response body downloads.
func probeProxy(ctx context.Context, upstream string) ProxyDiagnostic {
 start:=time.Now()
 out:=ProxyDiagnostic{Upstream:upstream,CheckedAt:start.UTC().Format(time.RFC3339Nano),State:"failed"}
 finish:=func(state,detail string)ProxyDiagnostic{out.State=state;out.Detail=detail;out.DurationMS=time.Since(start).Milliseconds();return out}
 target,err:=validate.Upstream(upstream);if err!=nil{return finish("invalid","目标地址无效")}
 ctx,cancel:=context.WithTimeout(ctx,5*time.Second);defer cancel()
 tr:=&http.Transport{DialContext:(&net.Dialer{Timeout:3*time.Second}).DialContext,TLSHandshakeTimeout:3*time.Second,ResponseHeaderTimeout:4*time.Second,DisableKeepAlives:true,MaxResponseHeaderBytes:64<<10}
 defer tr.CloseIdleConnections()
 client:=&http.Client{Transport:tr,CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse}}
 req,err:=http.NewRequestWithContext(ctx,http.MethodHead,target,nil);if err!=nil{return finish("invalid","目标地址无效")}
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
 if res.StatusCode>=500{return finish("http_error","上游已响应，但返回 5xx")}
 return finish("reachable","上游已响应 HTTP；不代表前端域名、登录或应用功能正常")
}

func(s *ProxyService) ProxyDiagnostics(ctx context.Context,id int64)([]ProxyDiagnostic,error){
 if _,err:=s.Get(ctx,id);err!=nil{return nil,err}
 rows,err:=s.db.QueryContext(ctx,"SELECT id,upstream,checked_at,state,duration_ms,http_status,detail FROM proxy_diagnostics WHERE rule_id=? ORDER BY id DESC LIMIT 5",id)
 if err!=nil{return nil,err};defer rows.Close()
 out:=[]ProxyDiagnostic{}
 for rows.Next(){var d ProxyDiagnostic;if err:=rows.Scan(&d.ID,&d.Upstream,&d.CheckedAt,&d.State,&d.DurationMS,&d.HTTPStatus,&d.Detail);err!=nil{return nil,err};out=append(out,d)}
 return out,rows.Err()
}
func(s *ProxyService) RunProxyDiagnostic(ctx context.Context,id int64)(ProxyDiagnostic,error){
 r,err:=s.Get(ctx,id);if err!=nil{return ProxyDiagnostic{},err}
 d:=probeProxy(ctx,r.Upstream)
 if ctx.Err()!=nil{return d,ctx.Err()}
 tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return d,err};defer tx.Rollback()
 result,err:=tx.ExecContext(ctx,"INSERT INTO proxy_diagnostics(rule_id,upstream,checked_at,state,duration_ms,http_status,detail) SELECT id,?,?,?,?,?,? FROM proxy_rules WHERE id=? AND upstream=?",d.Upstream,d.CheckedAt,d.State,d.DurationMS,d.HTTPStatus,d.Detail,id,r.Upstream)
 if err!=nil{return d,err}
 if n,err:=result.RowsAffected();err!=nil||n!=1{return d,fmt.Errorf("规则已变化，请重新检测")}
 d.ID,err=result.LastInsertId();if err!=nil{return d,err}
 if _,err=tx.ExecContext(ctx,"DELETE FROM proxy_diagnostics WHERE rule_id=? AND id NOT IN (SELECT id FROM proxy_diagnostics WHERE rule_id=? ORDER BY id DESC LIMIT 5)",id,id);err!=nil{return d,err}
 return d,tx.Commit()
}
