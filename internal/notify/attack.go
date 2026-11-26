package notify

import(
 "context"
 "fmt"
 "net/netip"
 "strings"
 "time"
 "github.com/fonu/fonu/internal/awareness"
 "github.com/fonu/fonu/internal/logstore"
)
const attackWindow=60*time.Second
const attackCooldown=5*time.Minute
type attackBucket struct{requests,probes,limited int; ips map[string]struct{}}
type attackState struct{start,lastScan,lastCC time.Time; hosts map[string]*attackBucket; scans map[string]int}
func(s *Service) cachedAccessConfig(ctx context.Context)(RuntimeConfig,error){
 s.accessConfigMu.Lock();defer s.accessConfigMu.Unlock()
 now:=time.Now();if now.Before(s.accessConfigUntil){return s.accessConfig,nil}
 cfg,err:=s.store.LoadRuntime(ctx);if err!=nil{return RuntimeConfig{},err}
 s.accessConfig=cfg;s.accessConfigUntil=now.Add(5*time.Second);return cfg,nil
}
func(s *Service) enqueueAlert(event,title,detail string){
 if s.alertSlots==nil{return}
 select{case s.alertSlots<-struct{}{}:
  go func(){defer func(){<-s.alertSlots}();ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel();s.Alert(ctx,event,title,detail)}()
 default: // bounded concurrency: never delay log ingestion during an attack.
 s.record(event,"","skipped","告警发送并发已满")
 }
}
func publicAttackIP(raw string)(string,bool){
 ip,err:=netip.ParseAddr(strings.TrimSpace(raw));if err!=nil{return "",false};ip=ip.Unmap()
 if !ip.IsGlobalUnicast()||ip.IsPrivate()||ip.IsLoopback()||ip.IsLinkLocalUnicast(){return "",false};return ip.String(),true
}
type attackNotice struct{event,title,detail string}
func(st *attackState) observe(entry logstore.AccessEntry,now time.Time,scan,cc bool)[]attackNotice{
 ip,ok:=publicAttackIP(entry.ClientIP);if !ok||(!scan&&!cc){return nil}
 if st.start.IsZero()||now.Sub(st.start)>=attackWindow||now.Before(st.start){st.start=now;st.hosts=map[string]*attackBucket{};st.scans=map[string]int{}}
 // Restrict attacker-controlled cardinality. Overflow events still contribute to a shared bucket.
 host:=strings.ToLower(entry.Domain);if len(host)>253||host==""{host="(其他入口)"}
 host=fmt.Sprintf("%s:%d",host,entry.ServerPort)
 bucket:=st.hosts[host];if bucket==nil{if len(st.hosts)>=256{host="(其他入口)";bucket=st.hosts[host]};if bucket==nil{bucket=&attackBucket{ips:map[string]struct{}{}};st.hosts[host]=bucket}}
 bucket.requests++
 if len(bucket.ips)<10{bucket.ips[ip]=struct{}{}}
 if entry.Status==429{bucket.limited++}
 var out []attackNotice
 if scan&&awareness.IsProbe(entry.Path,entry.UserAgent){
  bucket.probes++
  if _,ok:=st.scans[ip];ok||len(st.scans)<4096{st.scans[ip]++}
  if (st.scans[ip]>=5||bucket.probes>=20)&&now.Sub(st.lastScan)>=attackCooldown{
   st.lastScan=now
   out=append(out,attackNotice{EventScan,"疑似恶意扫描",fmt.Sprintf("入口 %s，来源 IP %s，当前 60 秒统计窗该 IP 命中 %d 次探测特征，入口命中 %d 次。请查看日志并确认是否需要封禁。",host,ip,st.scans[ip],bucket.probes)})
  }
 }
 if cc&&(((bucket.requests>=600&&len(bucket.ips)>=10)||bucket.requests>=1200)||bucket.limited>=100)&&now.Sub(st.lastCC)>=attackCooldown{
  st.lastCC=now
  out=append(out,attackNotice{EventCC,"疑似 CC / 高频异常访问",fmt.Sprintf("入口 %s，当前 60 秒统计窗请求 %d 次，来源公网 IP 至少 %d 个，429 限流 %d 次。请核对正常流量并启用入口限流或 CDN 防护。",host,bucket.requests,len(bucket.ips),bucket.limited)})
 }
 return out
}
func(s *Service) ObserveAttack(ctx context.Context,entry logstore.AccessEntry,now time.Time){
 if s==nil||s.store==nil{return}
 cfg,err:=s.cachedAccessConfig(ctx);if err!=nil||cfg.Type==""||(!cfg.OnScan&&!cfg.OnCC){return}
 s.securityMu.Lock();alerts:=s.securityState.observe(entry,now,cfg.OnScan,cfg.OnCC);s.securityMu.Unlock()
 for _,a:=range alerts{s.enqueueAlert(a.event,a.title,a.detail)}
}
