package backup

import (
 "context"
 "fmt"
 "os"
 "path/filepath"
 "regexp"
 "sort"
 "strings"
 "time"
)

const maxManualBackups=30
var localName=regexp.MustCompile(`^fonu-(manual|auto)-[0-9]{8}-[0-9]{6}-[a-zA-Z0-9]+[.]tar[.]gz$`)

type LocalBackup struct {
 Name string `json:"name"`
 Kind string `json:"kind"`
 Size int64 `json:"size"`
 CreatedAt time.Time `json:"created_at"`
}
func(s *Service)localDir()(string,error){
 base:=filepath.Join(s.dataDir,"backups")
 dir:=filepath.Join(base,"local")
 for _,p:=range []string{base,dir}{
  if err:=os.Mkdir(p,0700);err!=nil&&!os.IsExist(err){return "",err}
  info,err:=os.Lstat(p);if err!=nil{return "",err};if !info.IsDir()||info.Mode()&os.ModeSymlink!=0{return "",fmt.Errorf("备份目录无效")}
 }
 if err:=os.Chmod(dir,0700);err!=nil{return "",err}
 return dir,nil
}
func(s *Service)listLocked()([]LocalBackup,error){
 dir,err:=s.localDir();if err!=nil{return nil,err}
 entries,err:=os.ReadDir(dir);if err!=nil{return nil,err}
 out:=[]LocalBackup{}
 for _,e:=range entries{
  if !localName.MatchString(e.Name())||!e.Type().IsRegular(){continue}
  info,err:=e.Info();if err!=nil{return nil,err}
  if !info.Mode().IsRegular()||info.Size()==0{continue}
  kind:="manual";if strings.HasPrefix(e.Name(),"fonu-auto-"){kind="auto"}
  out=append(out,LocalBackup{Name:e.Name(),Kind:kind,Size:info.Size(),CreatedAt:info.ModTime().UTC()})
 }
 sort.Slice(out,func(i,j int)bool{if out[i].CreatedAt.Equal(out[j].CreatedAt){return out[i].Name>out[j].Name};return out[i].CreatedAt.After(out[j].CreatedAt)})
 return out,nil
}
func(s *Service) ListLocal()([]LocalBackup,error){s.mu.Lock();defer s.mu.Unlock();return s.listLocked()}
func(s *Service)localPath(name string)(string,error){
 if !localName.MatchString(name){return "",os.ErrNotExist}
 dir,err:=s.localDir();if err!=nil{return "",err}
 path:=filepath.Join(dir,name)
 info,err:=os.Lstat(path);if err!=nil{return "",err}
 if !info.Mode().IsRegular(){return "",os.ErrNotExist}
 return path,nil
}
func(s *Service)CreateLocal(ctx context.Context)(LocalBackup,error){
 s.mu.Lock();defer s.mu.Unlock()
 rows,err:=s.listLocked();if err!=nil{return LocalBackup{},err}
 n:=0;for _,row:=range rows{if row.Kind=="manual"{n++}}
 if n>=maxManualBackups{return LocalBackup{},fmt.Errorf("手动备份已达 30 份，请删除旧备份后再创建")}
 return s.createLocked(ctx,"manual")
}
func(s *Service)createLocked(ctx context.Context,kind string)(LocalBackup,error){
 dir,err:=s.localDir();if err!=nil{return LocalBackup{},err}
 path,err:=s.exportLocked(ctx);if err!=nil{return LocalBackup{},err}
 defer os.Remove(path)
 if err:=ctx.Err();err!=nil{return LocalBackup{},err}
 name:=strings.Replace(filepath.Base(path),"fonu-backup-","fonu-"+kind+"-",1)
 if !localName.MatchString(name){return LocalBackup{},fmt.Errorf("备份名称无效")}
 dest:=filepath.Join(dir,name)
 if _,err:=os.Lstat(dest);!os.IsNotExist(err){return LocalBackup{},fmt.Errorf("备份名称冲突")}
 if err:=os.Rename(path,dest);err!=nil{return LocalBackup{},err}
 info,err:=os.Stat(dest);if err!=nil{return LocalBackup{},err}
 return LocalBackup{Name:name,Kind:kind,Size:info.Size(),CreatedAt:info.ModTime().UTC()},nil
}
// AutoBackup runs at most once every 24 hours across restarts. Only automatic
// archives are pruned, and only after a valid replacement has been written.
func(s *Service)AutoBackup(ctx context.Context,keep int)error{
 if keep<1||keep>30{return fmt.Errorf("自动备份保留数量无效")}
 s.mu.Lock();defer s.mu.Unlock()
 rows,err:=s.listLocked();if err!=nil{return err}
 for _,row:=range rows{if row.Kind=="auto"&&time.Since(row.CreatedAt)<24*time.Hour{return nil}}
 if _,err:=s.createLocked(ctx,"auto");err!=nil{return err}
 rows,err=s.listLocked();if err!=nil{return err}
 count:=0
 for _,row:=range rows{
  if row.Kind!="auto"{continue};count++;if count<=keep{continue}
  if err:=ctx.Err();err!=nil{return err}
  path,err:=s.localPath(row.Name);if err!=nil{return err};if err:=os.Remove(path);err!=nil{return err}
 }
 return nil
}
func(s *Service) OpenLocal(name string)(*os.File,error){
 s.mu.Lock();defer s.mu.Unlock()
 path,err:=s.localPath(name);if err!=nil{return nil,err}
 f,err:=os.Open(path);if err!=nil{return nil,err}
 info,err:=f.Stat();if err!=nil||!info.Mode().IsRegular(){f.Close();return nil,os.ErrNotExist}
 return f,nil
}
func(s *Service) DeleteLocal(name string)error{s.mu.Lock();defer s.mu.Unlock();path,err:=s.localPath(name);if err!=nil{return err};return os.Remove(path)}
func(s *Service) RestoreLocal(ctx context.Context,name string)error{s.mu.Lock();defer s.mu.Unlock();path,err:=s.localPath(name);if err!=nil{return err};return s.restoreLocked(ctx,path)}
