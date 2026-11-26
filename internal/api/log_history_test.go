package api

import (
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "github.com/fonu/fonu/internal/config"
)

func TestHistorySummaryAndFilteredPages(t *testing.T){
 cfg:=config.Config{DataDir:t.TempDir()}
 if err:=os.MkdirAll(cfg.LogsDir(),0700);err!=nil{t.Fatal(err)}
 put:=func(name,content string){t.Helper();if err:=os.WriteFile(filepath.Join(cfg.LogsDir(),name),[]byte(content),0600);err!=nil{t.Fatal(err)}}
 put("access.log",strings.Join([]string{
 "2026-11-25T08:00:00Z example.com 443 GET /old 403 0.001 192.0.2.1 - 30 100 \"Browser\"",
 "2026-11-26T08:00:00Z example.com 443 GET /recent 200 0.001 192.0.2.1 - 30 100 \"Browser\"",
 "2026-11-26T09:00:00Z example.com 443 GET /new 403 0.001 192.0.2.1 - 30 100 \"Browser\"",
 "2026-11-26T10:00:00Z example.com 443 GET /different 403 0.001 192.0.2.2 - 30 100 \"Browser\"",
 },"\n")+"\n")
 put("access.log.1","2026-11-24T08:00:00Z example.com 443 GET /archive 403 0.001 192.0.2.1 - 30 100 \"Browser\"\n")
 put("app.log","{\"time\":\"2026-11-26T08:00:00Z\",\"level\":\"INFO\",\"module\":\"SYSTEM\",\"msg\":\"started\"}\n{\"time\":\"2026-11-26T08:00:00Z\",\"level\":\"INFO\",\"module\":\"HTTP\",\"msg\":\"noise\"}\n")
 put("error.log","2026/11/26 08:00:00 [error] upstream failed\n")
 put("frpc.log","2026-11-26 08:00:00 [I] login success\n")
 h:=NewLogsHandler(cfg,nil,nil)
 rec:=httptest.NewRecorder();h.Summary(rec,httptest.NewRequest(http.MethodGet,"/api/logs/summary",nil))
 var summary historySummary
 if rec.Code!=200||json.Unmarshal(rec.Body.Bytes(),&summary)!=nil{t.Fatalf("summary: %d %s",rec.Code,rec.Body.String())}
 for source,want:=range map[string]int{"system":1,"access":5,"nginx":1,"frp":1}{got:=summary.Sources[source];if got.Count!=want||got.Error!=""{t.Fatalf("%s: %+v",source,got)}}
 call:=func(query string) historyPage {t.Helper();w:=httptest.NewRecorder();h.History(w,httptest.NewRequest(http.MethodGet,"/api/logs/history?"+query,nil));var page historyPage;if w.Code!=200||json.Unmarshal(w.Body.Bytes(),&page)!=nil{t.Fatalf("history: %d %s",w.Code,w.Body.String())};return page}
 filter:="source=access&ip=192.0.2.1&status=403&limit=1"
 first:=call(filter);second:=call(filter+"&offset=1");third:=call(filter+"&offset=2")
 for i,v:=range []struct{page historyPage;path string;more bool}{{first,"/new",true},{second,"/old",true},{third,"/archive",false}}{if len(v.page.Rows)!=1||v.page.Rows[0].(map[string]any)["path"]!=v.path||v.page.HasMore!=v.more{t.Fatalf("page %d: %+v",i,v.page)}}
 bounded:=call("source=access&from=2026-11-26T00:00:00Z&to=2026-11-26T09:30:00Z&status=403")
 if len(bounded.Rows)!=1||bounded.Rows[0].(map[string]any)["path"]!="/new"{t.Fatalf("date filter: %+v",bounded)}
 exported:=call("source=access&kind=intercept&export=1&limit=10000")
 if len(exported.Rows)!=4||exported.HasMore{t.Fatalf("export: %+v",exported)}
 w:=httptest.NewRecorder();h.History(w,httptest.NewRequest(http.MethodGet,"/api/logs/history?source=../../etc/passwd",nil));if w.Code!=400{t.Fatalf("invalid source accepted: %d",w.Code)}
}
