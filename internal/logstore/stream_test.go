package logstore

import (
 "context"
 "errors"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

type streamCapture struct{chunks chan string}
func(c streamCapture)Write(p []byte)(int,error){c.chunks<-string(p);return len(p),nil}

func TestStreamRotationAndPartialLine(t *testing.T){
 path:=filepath.Join(t.TempDir(),"access.log")
 if err:=os.WriteFile(path,[]byte("old\npart"),0600);err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background());defer cancel()
 capture:=streamCapture{make(chan string,32)}
 done:=make(chan error,1)
 go func(){done<-StreamFile(ctx,path,capture,func()error{return nil},1,nil)}()
 wait:=func(want string){
  t.Helper()
  timeout:=time.NewTimer(3*time.Second);defer timeout.Stop()
  for{select{case v:=<-capture.chunks:if strings.Contains(v,want){return};case err:=<-done:t.Fatalf("stream ended: %v",err);case <-timeout.C:t.Fatalf("missing %q",want)}}
 }
 wait("data: old\n")
 f,err:=os.OpenFile(path,os.O_APPEND|os.O_WRONLY,0600);if err!=nil{t.Fatal(err)}
 if _,err:=f.WriteString("ial\n");err!=nil{t.Fatal(err)};f.Close()
 wait("data: partial\n")
 if err:=os.Rename(path,path+".1");err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(path,[]byte("rotated\n"),0600);err!=nil{t.Fatal(err)}
 wait("data: rotated\n")
 cancel()
 select{case err:=<-done:if !errors.Is(err,context.Canceled){t.Fatal(err)};case <-time.After(time.Second):t.Fatal("stream did not stop")}
}

func TestTailSkipsOversizedLine(t *testing.T){
 path:=filepath.Join(t.TempDir(),"access.log")
 raw:="before\n"+strings.Repeat("x",maxLogLineBytes*2)+"\nafter\n"
 if err:=os.WriteFile(path,[]byte(raw),0600);err!=nil{t.Fatal(err)}
 lines,err:=TailLines(path,10)
 if err!=nil||len(lines)!=2||lines[0]!="before"||lines[1]!="after"{t.Fatalf("unexpected tail count=%d err=%v",len(lines),err)}
}
