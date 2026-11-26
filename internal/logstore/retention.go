package logstore

import (
 "context"
 "os"
 "time"
)

// PruneHistoryArchives removes only known, rotated regular files. Never
// truncate or delete the active log, even when it has not changed recently.
func PruneHistoryArchives(ctx context.Context,paths []string,cutoff time.Time)error{
 for _,path:=range paths{
  if err:=ctx.Err();err!=nil{return err}
  files,err:=HistoryFiles(path);if err!=nil{return err}
  for _,name:=range files{
   if err:=ctx.Err();err!=nil{return err}
   if name==path{continue}
   info,err:=os.Lstat(name);if os.IsNotExist(err){continue};if err!=nil{return err}
   if !info.Mode().IsRegular()||!info.ModTime().Before(cutoff){continue}
   if err:=os.Remove(name);err!=nil&&!os.IsNotExist(err){return err}
  }
 }
 return nil
}
