package api

import (
 "net/http"
 "context"
 "fmt"
 "os/exec"
 "path/filepath"
 "github.com/fonu/fonu/internal/nginx"
 "github.com/fonu/fonu/internal/version"
 "github.com/fonu/fonu/internal/logstore"
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
 out["versions"]=h.versions(r.Context())
 processes:=map[string]processGroup{}
 if p,ok:=readRuntimeProcess(os.Getpid());ok{processes["fonu"]=groupProcesses([]runtimeProcess{p})}
 processes["nginx"]=readProcessGroup(h.cfg.NginxPIDFile,h.cfg.NginxBin,true)
 processes["frpc"]=readProcessGroup(h.cfg.FrpPIDFile,h.cfg.FrpcBin,false)
 out["processes"]=processes
 if h.proxySvc!=nil{out["nginx_apply"]=h.proxySvc.NginxApplyStatus()}
 if lines,err:=logstore.TailLines(filepath.Join(h.cfg.LogsDir(),"error.log"),5);err==nil{
  for i,line:=range lines{if len(line)>2048{lines[i]=line[:2048]+"…"}}
  out["nginx_errors"]=lines
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

type runtimeProcess struct{
 PID int `json:"pid"`
 Start uint64 `json:"start"`
 Ticks uint64 `json:"ticks"`
 RSS uint64 `json:"rss_bytes"`
 FD *int `json:"fd_count,omitempty"`
}
type processGroup struct{
 Running bool `json:"running"`
 RSS uint64 `json:"rss_bytes"`
 Processes []runtimeProcess `json:"processes"`
}
func groupProcesses(ps []runtimeProcess)processGroup{
 g:=processGroup{Running:len(ps)>0,Processes:ps}
 for _,p:=range ps{g.RSS+=p.RSS}
 return g
}
func parseProcessStat(raw string)(ticks,start uint64,ok bool){
 end:=strings.LastIndexByte(raw,')');if end<0{return 0,0,false}
 fields:=strings.Fields(raw[end+1:]);if len(fields)<20||fields[0]=="Z"{return 0,0,false}
 u,e:=strconv.ParseUint(fields[11],10,64);if e!=nil{return 0,0,false}
 s,e:=strconv.ParseUint(fields[12],10,64);if e!=nil{return 0,0,false}
 start,e=strconv.ParseUint(fields[19],10,64);return u+s,start,e==nil
}
func readRuntimeProcess(pid int)(runtimeProcess,bool){
 base:=fmt.Sprintf("/proc/%d",pid)
 raw,err:=os.ReadFile(base+"/stat");if err!=nil{return runtimeProcess{},false}
 ticks,start,ok:=parseProcessStat(string(raw));if !ok{return runtimeProcess{},false}
 p:=runtimeProcess{PID:pid,Start:start,Ticks:ticks}
 if raw,err:=os.ReadFile(base+"/status");err==nil{if n,ok:=procKB(string(raw),"VmRSS:");ok{p.RSS=n*1024}}
 if entries,err:=os.ReadDir(base+"/fd");err==nil{n:=len(entries);p.FD=&n}
 raw,err=os.ReadFile(base+"/stat");if err!=nil{return runtimeProcess{},false}
 _,again,ok:=parseProcessStat(string(raw));return p,ok&&again==start
}
func readProcessGroup(pidfile,bin string,children bool)processGroup{
 empty:=groupProcesses([]runtimeProcess{})
 pid,err:=nginx.ReadPID(pidfile);if err!=nil||pid<=1{return empty}
 path,err:=exec.LookPath(bin);if err!=nil{return empty}
 expected,err:=os.Stat(path);if err!=nil{return empty}
 matches:=func(pid int)bool{actual,err:=os.Stat(fmt.Sprintf("/proc/%d/exe",pid));return err==nil&&os.SameFile(expected,actual)}
 if !matches(pid){return empty}
 master,ok:=readRuntimeProcess(pid);if !ok{return empty}
 ps:=[]runtimeProcess{master}
 if children{
  raw,err:=os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children",pid,pid))
  if err==nil{for _,field:=range strings.Fields(string(raw)){
   child,e:=strconv.Atoi(field);if e!=nil||!matches(child){continue}
   if p,ok:=readRuntimeProcess(child);ok{ps=append(ps,p)}
   if len(ps)>=256{break}
  }}
 }
 return groupProcesses(ps)
}
type versionOutput struct{data []byte}
func(v *versionOutput)Write(p []byte)(int,error){n:=len(p);if remaining:=512-len(v.data);remaining>0{if len(p)>remaining{p=p[:remaining]};v.data=append(v.data,p...)};return n,nil}
func readBinaryVersion(ctx context.Context,bin,arg string)string{
 if strings.TrimSpace(bin)==""{return ""}
 ctx,cancel:=context.WithTimeout(ctx,2*time.Second);defer cancel()
 cmd:=exec.CommandContext(ctx,bin,arg)
 cmd.WaitDelay=time.Second
 var output versionOutput
 cmd.Stdout=&output;cmd.Stderr=&output
 if err:=cmd.Run();err!=nil{return ""}
 line,_,_:=strings.Cut(strings.TrimSpace(string(output.data)),"\n")
 return strings.TrimSpace(line)
}
func(h *StatusHandler) versions(ctx context.Context)map[string]string{
 h.runtimeMu.Lock();defer h.runtimeMu.Unlock()
 if h.runtimeVersions!=nil&&time.Since(h.runtimeVersionAt)<10*time.Minute{return h.runtimeVersions}
 out:=map[string]string{"fonu":version.Version,"go":runtime.Version(),"nginx":readBinaryVersion(ctx,h.cfg.NginxBin,"-v"),"frpc":readBinaryVersion(ctx,h.cfg.FrpcBin,"--version")}
 h.runtimeVersions=out;h.runtimeVersionAt=time.Now();return out
}
