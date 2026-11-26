package logstore

import("context";"os";"path/filepath";"testing";"time")

func TestPruneHistoryArchives(t *testing.T){
 dir:=t.TempDir();now:=time.Now();old:=now.AddDate(0,0,-40)
 for _,name:=range []string{"app.log","app.log.1","app.log.2.gz","unrelated.log"}{
  path:=filepath.Join(dir,name);if err:=os.WriteFile(path,[]byte("log"),0600);err!=nil{t.Fatal(err)}
  if name!="app.log.2.gz"{if err:=os.Chtimes(path,old,old);err!=nil{t.Fatal(err)}}
 }
 if err:=PruneHistoryArchives(context.Background(),[]string{filepath.Join(dir,"app.log")},now.AddDate(0,0,-30));err!=nil{t.Fatal(err)}
 for _,name:=range []string{"app.log","app.log.2.gz","unrelated.log"}{if _,err:=os.Stat(filepath.Join(dir,name));err!=nil{t.Fatalf("%s: %v",name,err)}}
 if _,err:=os.Stat(filepath.Join(dir,"app.log.1"));!os.IsNotExist(err){t.Fatalf("old archive still present: %v",err)}
}
