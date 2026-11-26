package logstore

import (
 "bufio"
 "bytes"
 "compress/gzip"
 "context"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "regexp"
 "sort"
 "strings"
 "time"
)

// HistoryFiles restricts discovery to the requested, fixed log family.
// Current file is newest; archives are ordered by last modification.
func HistoryFiles(path string) ([]string,error) {
 dir, base := filepath.Dir(path), filepath.Base(path)
 entries,err:=os.ReadDir(dir)
 if os.IsNotExist(err) {return nil,nil};if err!=nil{return nil,err}
 stem:=strings.TrimSuffix(base,".log")
 pattern:=regexp.MustCompile("^"+regexp.QuoteMeta(stem)+"-[0-9][0-9T:.Z+_-]*\\.log(?:\\.gz)?$|^"+regexp.QuoteMeta(base)+"\\.[0-9]+(?:\\.gz)?$")
 type archive struct {path string; at time.Time}
 var archives []archive
 var current string
 for _,entry:=range entries {
  if entry.Type()&os.ModeSymlink!=0 || entry.IsDir(){continue}
  name:=entry.Name()
  if name!=base && !pattern.MatchString(name){continue}
  info,err:=entry.Info();if err!=nil{return nil,err};if !info.Mode().IsRegular(){continue}
  full:=filepath.Join(dir,name)
  if name==base {current=full;continue}
  archives=append(archives,archive{full,info.ModTime()})
 }
 sort.Slice(archives,func(i,j int)bool{if archives[i].at.Equal(archives[j].at){return archives[i].path>archives[j].path};return archives[i].at.After(archives[j].at)})
 out:=make([]string,0,len(archives)+1)
 if current!=""{out=append(out,current)}
 for _,a:=range archives{out=append(out,a.path)}
 return out,nil
}

// VisitHistory visits newest records first, holding at most a chunk and a line
// for regular files. Compressed archives are capped at 64 MiB decompressed.
func VisitHistory(ctx context.Context,path string,visit func(string)bool)error{
 files,err:=HistoryFiles(path);if err!=nil{return err}
 for _,name:=range files {
  if err:=ctx.Err();err!=nil{return err}
  stop,err:=visitHistoryFile(ctx,name,visit)
  if err!=nil{return err};if stop{return nil}
 }
 return nil
}

const historyLineLimit=1024*1024
const historyArchiveLimit=64*1024*1024
func visitHistoryFile(ctx context.Context,path string,visit func(string)bool)(bool,error){
 f,err:=os.Open(path);if err!=nil{return false,err};defer f.Close()
 info,err:=f.Stat();if err!=nil{return false,err}
 if !info.Mode().IsRegular(){return false,fmt.Errorf("not a regular log file")}
 if strings.HasSuffix(path,".gz"){
  gz,err:=gzip.NewReader(f);if err!=nil{return false,err};defer gz.Close()
  // Check cancellation while inflating, rather than allocating an unbounded archive.
  var data bytes.Buffer
  reader:=bufio.NewReaderSize(gz,64*1024)
  buf:=make([]byte,64*1024)
  for{if err:=ctx.Err();err!=nil{return false,err};n,e:=reader.Read(buf);if data.Len()+n>historyArchiveLimit{return false,fmt.Errorf("log archive exceeds size limit")};data.Write(buf[:n]);if e==io.EOF{break};if e!=nil{return false,e}}
  b:=data.Bytes()
  for len(b)>0{if err:=ctx.Err();err!=nil{return false,err};i:=bytes.LastIndexByte(b,'\n');line:=b[i+1:];if len(line)>historyLineLimit{return false,fmt.Errorf("log line exceeds size limit")};if v:=strings.TrimSpace(string(line));v!=""&&!visit(v){return true,nil};if i<0{break};b=b[:i]}
  return false,nil
 }
 offset:=info.Size();var pending []byte
 for offset>0{
  if err:=ctx.Err();err!=nil{return false,err}
  n:=int64(tailReadChunk);if n>offset{n=offset};offset-=n
  chunk:=make([]byte,int(n));if _,err:=f.ReadAt(chunk,offset);err!=nil{return false,err}
  b:=append(chunk,pending...);pending=nil
  for len(b)>0{if err:=ctx.Err();err!=nil{return false,err};i:=bytes.LastIndexByte(b,'\n');if i<0{pending=b;break};line:=b[i+1:];if len(line)>historyLineLimit{return false,fmt.Errorf("log line exceeds size limit")};if v:=strings.TrimSpace(string(line));v!=""&&!visit(v){return true,nil};b=b[:i]}
  if len(pending)>historyLineLimit{return false,fmt.Errorf("log line exceeds size limit")}
 }
 if v:=strings.TrimSpace(string(pending));v!=""&&!visit(v){return true,nil}
 return false,nil
}

// ParseHistorySystem retains existing UI noise filtering.
func ParseHistorySystem(line string)(SystemEntry,bool){
 entry,ok:=parseSystem(line)
 return entry,ok&&!isNoiseSystemEntry(entry)
}
