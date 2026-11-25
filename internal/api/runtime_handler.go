package api

import (
 "net/http"
 "os"
 "runtime"
 "strconv"
 "strings"
 "time"
)

// Runtime exposes inexpensive read-only snapshots; no shell commands or sampling sleeps.
func (h *StatusHandler) Runtime(w http.ResponseWriter, r *http.Request) {
 var mem runtime.MemStats
 runtime.ReadMemStats(&mem)
 out := map[string]any{"heap_bytes":mem.HeapAlloc,"goroutines":runtime.NumGoroutine()}
 started,err:=time.Parse(time.RFC3339,h.startedAt)
 if err==nil {out["uptime_seconds"]=int64(time.Since(started).Seconds())}
 if data,err:=os.ReadFile("/proc/self/status");err==nil {
  if n,ok:=procKB(string(data),"VmRSS:");ok{out["rss_bytes"]=n*1024}
 }
 if data,err:=os.ReadFile("/proc/meminfo");err==nil {
  if n,ok:=procKB(string(data),"MemTotal:");ok{out["host_memory_total"]=n*1024}
  if n,ok:=procKB(string(data),"MemAvailable:");ok{out["host_memory_available"]=n*1024}
 }
 if data,err:=os.ReadFile("/proc/stat");err==nil {
  if total,idle,ok:=procCPU(string(data));ok{out["host_cpu_total"]=total;out["host_cpu_idle"]=idle}
 }
 w.Header().Set("Cache-Control","no-store")
 writeJSON(w,http.StatusOK,out)
}
func procKB(data,key string)(uint64,bool){
 for _,line:=range strings.Split(data,"\n"){
  fields:=strings.Fields(line)
  if len(fields)>=3 && fields[0]==key && fields[2]=="kB" {n,err:=strconv.ParseUint(fields[1],10,64);return n,err==nil}
 }
 return 0,false
}
func procCPU(data string)(total,idle uint64,ok bool){
 line,_,_:=strings.Cut(data,"\n")
 fields:=strings.Fields(line)
 if len(fields)<9 || fields[0]!="cpu"{return 0,0,false}
 for i:=1;i<=8;i++{n,err:=strconv.ParseUint(fields[i],10,64);if err!=nil{return 0,0,false};total+=n;if i==4||i==5{idle+=n}}
 return total,idle,total>0
}
