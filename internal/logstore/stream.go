package logstore

import (
 "bufio"
 "context"
 "errors"
 "fmt"
 "io"
 "os"
 "strings"
 "time"
)

type LineFilter func(line string) bool

func StreamFile(ctx context.Context,path string,w io.Writer,flush func()error,tail int,filter LineFilter)error{
 if tail>1000{tail=1000}
 emit:=func(line string)error{
  line=strings.TrimRight(line,"\r\n")
  if strings.TrimSpace(line)=="" || (filter!=nil&&!filter(line)){return nil}
  line=strings.ReplaceAll(line,"\r"," ")
  if _,err:=fmt.Fprintf(w,"event: log\ndata: %s\n\n",line);err!=nil{return err}
  return flush()
 }
 var file *os.File
 defer func(){if file!=nil{_ = file.Close()}}()
 var reader *bufio.Reader
 var offset int64
 var partial []byte
 dropping:=false
 initial:=true
 ticker:=time.NewTicker(250*time.Millisecond);defer ticker.Stop()
 heartbeat:=time.NewTicker(15*time.Second);defer heartbeat.Stop()
 open:=func()error{
  f,err:=os.Open(path);if err!=nil{return err}
  info,err:=f.Stat();if err!=nil{f.Close();return err}
  if !info.Mode().IsRegular(){f.Close();return fmt.Errorf("日志不是普通文件")}
  file=f;offset=0;partial=nil;dropping=false
  if initial{
   size:=info.Size()
   end:=size
   if size>0{
    var last [1]byte
    if _,err:=file.ReadAt(last[:],size-1);err!=nil{return err}
    if last[0]!='\n'{
     start:=size-int64(maxLogLineBytes);if start<0{start=0}
     buf:=make([]byte,size-start)
     n,err:=file.ReadAt(buf,start);if err!=nil&&err!=io.EOF{return err}
     idx:=strings.LastIndexByte(string(buf[:n]),'\n')
     if idx>=0{end=start+int64(idx)+1}else if start==0{end=0}else{dropping=true}
    }
   }
   if tail>0{
    lines,err:=tailLinesAt(file,end,tail);if err!=nil{return err}
    for _,line:=range lines{if err:=ctx.Err();err!=nil{return err};if err:=emit(line);err!=nil{return err}}
   }
   offset=end
   if _,err:=file.Seek(offset,io.SeekStart);err!=nil{return err}
   initial=false
  }
  reader=bufio.NewReaderSize(file,32<<10)
  return nil
 }
 if err:=open();err!=nil&&!os.IsNotExist(err){return err}
 if _,err:=io.WriteString(w,": connected\n\n");err!=nil{return err}
 if err:=flush();err!=nil{return err}
 for {
  if err:=ctx.Err();err!=nil{return err}
  if file!=nil{
   part,err:=reader.ReadSlice('\n')
   offset+=int64(len(part))
   if !dropping{
    if len(part)>maxLogLineBytes-len(partial){partial=nil;dropping=true}else{partial=append(partial,part...)}
   }
   if err==nil{
    if !dropping{if e:=emit(string(partial));e!=nil{return e}}
    partial=partial[:0];dropping=false
    continue
   }
   if errors.Is(err,bufio.ErrBufferFull){continue}
   if err!=io.EOF{return err}
   current,e:=os.Stat(path)
   if e==nil{
    old,e:=file.Stat();if e!=nil{return e}
    if !os.SameFile(old,current){
     _=file.Close();file=nil;reader=nil;partial=nil;dropping=false
    }else if current.Size()<offset{
     if _,e:=file.Seek(0,io.SeekStart);e!=nil{return e}
     reader.Reset(file);offset=0;partial=nil;dropping=false
    }
   }else if !os.IsNotExist(e){return e}
  }
  if file==nil{
   if err:=open();err!=nil&&!os.IsNotExist(err){return err}
   if file!=nil{continue}
  }
  select{
  case <-ctx.Done():return ctx.Err()
  case <-ticker.C:
  case <-heartbeat.C:
   if _,err:=io.WriteString(w,": heartbeat\n\n");err!=nil{return err}
   if err:=flush();err!=nil{return err}
  }
 }
}

func trimSSE(line string)string{return strings.TrimRight(line,"\r\n")}
