package logstore

import (
 "compress/gzip"
 "context"
 "os"
 "path/filepath"
 "reflect"
 "strings"
 "testing"
)

func TestHistoryCurrentAndRotatedArchives(t *testing.T){
 dir:=t.TempDir();path:=filepath.Join(dir,"app.log")
 long:=strings.Repeat("x",70000)
 if err:=os.WriteFile(path,[]byte("current-1\n"+long+"\ncurrent-3\n"),0600);err!=nil{t.Fatal(err)}
 archive,err:=os.Create(filepath.Join(dir,"app-2026-11-25T10-00-00.000.log.gz"));if err!=nil{t.Fatal(err)}
 gz:=gzip.NewWriter(archive)
 if _,err:=gz.Write([]byte("old-1\nold-2\n"));err!=nil{t.Fatal(err)}
 if err:=gz.Close();err!=nil{t.Fatal(err)};if err:=archive.Close();err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,"unrelated.log"),[]byte("never\n"),0600);err!=nil{t.Fatal(err)}
 var got []string
 if err:=VisitHistory(context.Background(),path,func(s string)bool{got=append(got,s);return true});err!=nil{t.Fatal(err)}
 want:=[]string{"current-3",long,"current-1","old-2","old-1"}
 if !reflect.DeepEqual(got,want){t.Fatalf("unexpected order or line count: %d",len(got))}
 got=nil
 if err:=VisitHistory(context.Background(),path,func(s string)bool{got=append(got,s);return len(got)<2});err!=nil{t.Fatal(err)}
 if len(got)!=2||got[0]!="current-3"||got[1]!=long{t.Fatal("early stop failed")}
 ctx,cancel:=context.WithCancel(context.Background());cancel()
 if err:=VisitHistory(ctx,path,func(string)bool{return true});err==nil{t.Fatal("expected cancellation")}
}
