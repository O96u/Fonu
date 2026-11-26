package api

import (
 "context"
 "net/http"
 "path/filepath"
 "regexp"
 "strconv"
 "strings"
 "time"
 "github.com/fonu/fonu/internal/awareness"
 "github.com/fonu/fonu/internal/logstore"
 "github.com/fonu/fonu/internal/proxy"
)

var historySources=[]string{"system","access","nginx","frp"}
type logCount struct {Count int `json:"count"`; Error string `json:"error,omitempty"`}
type historySummary struct {Sources map[string]logCount `json:"sources"`; UpdatedAt time.Time `json:"updated_at"`}
type historyPage struct {Rows []any `json:"rows"`; HasMore bool `json:"has_more"`; Offset int `json:"offset"`; Limit int `json:"limit"`}
type historyRecord struct {value any; at time.Time; level,search,service,ip,kind string;status int}
var historyLineTime=regexp.MustCompile(`^(\d{4}[-/]\d{2}[-/]\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?)\s+\[([^\]]+)\]`)
func (h *LogsHandler) historyPath(source string)string{
 switch source{case "system":return filepath.Join(h.cfg.LogsDir(),"app.log");case "access":return filepath.Join(h.cfg.LogsDir(),"access.log");case "nginx":return filepath.Join(h.cfg.LogsDir(),"error.log");case "frp":return h.cfg.FrpLogPath()};return ""
}
func historyTime(s string)time.Time{
 if t,e:=time.Parse(time.RFC3339Nano,s);e==nil{return t}
 s=strings.ReplaceAll(s,"/","-")
 if t,e:=time.ParseInLocation("2006-01-02 15:04:05",s,time.Local);e==nil{return t}
 return time.Time{}
}
func historyLevel(s string)string{
 switch strings.ToUpper(s){case "E","ERROR","EMERG","ALERT","CRIT":return "ERROR";case "W","WARN","WARNING":return "WARN";case "I","INFO","NOTICE":return "INFO";case "D","DEBUG":return "DEBUG"};return "UNKNOWN"
}
func parseHistoryRecord(source,line string,bindings proxy.BindingIndex) (historyRecord,bool){
 switch source{
 case "access":
  e,ok:=logstore.ParseAccess(line);if !ok{return historyRecord{},false}
  service:=proxy.ResolveServiceLabel(e.Domain,e.ServerPort,e.Upstream,bindings)
  kind,reason:=awareness.ClassifyAccess(e)
  row:=awareness.EnrichedAccessLog{AccessEntry:e,Service:service,AwarenessKind:string(kind)}
  if kind!=awareness.KindAccess{row.BlockReason=reason}
  return historyRecord{value:row,at:historyTime(e.Time),search:strings.Join([]string{service,e.Domain,e.Path,e.ClientIP,e.Method,e.Upstream,e.UserAgent,reason,strconv.Itoa(e.Status)}," "),service:service+" "+e.Domain,ip:e.ClientIP,kind:string(kind),status:e.Status},true
 case "system":
  e,ok:=logstore.ParseHistorySystem(line);if !ok{return historyRecord{},false}
  return historyRecord{value:e,at:historyTime(e.Time),level:historyLevel(e.Level),search:e.Module+" "+e.Message,service:e.Module},true
 default:
  r:=historyRecord{value:line,search:line,service:line,level:"UNKNOWN"}
  if m:=historyLineTime.FindStringSubmatch(line);len(m)==3{r.at=historyTime(m[1]);r.level=historyLevel(m[2])}
  return r,true
 }
}
func (h *LogsHandler) acquireHistory(ctx context.Context)bool{
 select{case h.historyGate<-struct{}{}:return true;case <-ctx.Done():return false}
}
func (h *LogsHandler) Summary(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 h.historyMu.Lock()
 cached:=h.historySummary
 h.historyMu.Unlock()
 if r.URL.Query().Get("force")!="1"&&!cached.UpdatedAt.IsZero()&&time.Since(cached.UpdatedAt)<30*time.Second{writeJSON(w,http.StatusOK,cached);return}
 ctx,cancel:=context.WithTimeout(r.Context(),25*time.Second);defer cancel()
 if !h.acquireHistory(ctx){writeError(r,w,http.StatusServiceUnavailable,"日志统计繁忙，请重试");return};defer func(){<-h.historyGate}()
 h.historyMu.Lock();cached=h.historySummary;h.historyMu.Unlock()
 if r.URL.Query().Get("force")!="1"&&!cached.UpdatedAt.IsZero()&&time.Since(cached.UpdatedAt)<30*time.Second{writeJSON(w,http.StatusOK,cached);return}
 result:=historySummary{Sources:map[string]logCount{}}
 for _,source:=range historySources{
  count:=0
  err:=logstore.VisitHistory(ctx,h.historyPath(source),func(line string)bool{
   if source=="access"{if _,ok:=logstore.ParseAccess(line);ok{count++}}else if source=="system"{if _,ok:=logstore.ParseHistorySystem(line);ok{count++}}else{count++};return true
  })
  if err!=nil{result.Sources[source]=logCount{Error:"统计未完成，请刷新重试"}}else{result.Sources[source]=logCount{Count:count}}
 }
 result.UpdatedAt=time.Now()
 h.historyMu.Lock();h.historySummary=result;h.historyMu.Unlock()
 writeJSON(w,http.StatusOK,result)
}
func (h *LogsHandler) History(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Cache-Control","no-store")
 q:=r.URL.Query();source:=q.Get("source");path:=h.historyPath(source)
 if path==""{writeError(r,w,http.StatusBadRequest,"未知日志类型");return}
 offset,limit:=queryInt(r,"offset",0),queryInt(r,"limit",20)
 maxLimit:=100
 if q.Get("export")=="1"{maxLimit=10000}
 if offset<0||limit<1||limit>maxLimit{writeError(r,w,http.StatusBadRequest,"分页参数无效");return}
 if q.Get("export")=="1"&&offset!=0{writeError(r,w,http.StatusBadRequest,"导出偏移无效");return}
 parseBound:=func(key string)(time.Time,error){if q.Get(key)==""{return time.Time{},nil};return time.Parse(time.RFC3339Nano,q.Get(key))}
 from,e1:=parseBound("from");to,e2:=parseBound("to");snapshot,e3:=parseBound("snapshot")
 if e1!=nil||e2!=nil||e3!=nil||(!from.IsZero()&&!to.IsZero()&&from.After(to)){writeError(r,w,http.StatusBadRequest,"时间范围无效");return}
 kw,level,svc,ip,kind:=strings.ToLower(strings.TrimSpace(q.Get("keyword"))),strings.ToUpper(q.Get("level")),strings.ToLower(strings.TrimSpace(q.Get("service"))),strings.TrimSpace(q.Get("ip")),q.Get("kind")
 status:=queryInt(r,"status",0)
 if len(kw)>512||len(svc)>256||len(ip)>128{writeError(r,w,http.StatusBadRequest,"筛选条件过长");return}
 ctx,cancel:=context.WithTimeout(r.Context(),25*time.Second);defer cancel()
 if !h.acquireHistory(ctx){writeError(r,w,http.StatusServiceUnavailable,"日志查询繁忙，请重试");return};defer func(){<-h.historyGate}()
 var bindings proxy.BindingIndex
 if source=="access"&&h.proxySvc!=nil{if rules,err:=h.proxySvc.List(ctx);err==nil{bindings=proxy.BuildBindingIndex(rules)}}
 result:=historyPage{Rows:make([]any,0,limit),Offset:offset,Limit:limit}
 matched:=0
 resultBytes:=0
 oversized:=false
 err:=logstore.VisitHistory(ctx,path,func(line string)bool{
  row,ok:=parseHistoryRecord(source,line,bindings);if !ok{return true}
  if !snapshot.IsZero()&&!row.at.IsZero()&&row.at.After(snapshot){return true}
  if (!from.IsZero()||!to.IsZero())&&row.at.IsZero(){return true}
  if (!from.IsZero()&&row.at.Before(from))||(!to.IsZero()&&row.at.After(to)){return true}
  if kw!=""&&!strings.Contains(strings.ToLower(row.search),kw){return true}
  if level!=""&&row.level!=level{return true}
  if svc!=""&&!strings.Contains(strings.ToLower(row.service),svc){return true}
  if ip!=""&&!strings.EqualFold(ip,row.ip){return true}
  if status!=0&&row.status!=status{return true}
  if kind!=""&&!awareness.MatchKindFilter(awareness.EventKind(row.kind),kind){return true}
  if matched<offset{matched++;return true}
  if len(result.Rows)>=limit{result.HasMore=true;return false}
  resultBytes+=len(line)
  if resultBytes>16*1024*1024{oversized=true;return false}
  result.Rows=append(result.Rows,row.value);return true
 })
 if err!=nil{writeError(r,w,http.StatusInternalServerError,"读取历史日志失败或超时，请缩小范围后重试");return}
 if oversized{writeError(r,w,http.StatusRequestEntityTooLarge,"日志结果过大，请缩小时间范围或减少每页条数");return}
 if source=="access"&&h.geo!=nil&&q.Get("export")!="1"{
  cache:=map[string]string{}
  for i,item:=range result.Rows{if row,ok:=item.(awareness.EnrichedAccessLog);ok{row.Geo=h.geo.LookupCached(row.ClientIP,cache);result.Rows[i]=row}}
 }
 writeJSON(w,http.StatusOK,result)
}
