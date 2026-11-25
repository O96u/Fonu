package acme

import (
 "crypto/tls"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 certstore "github.com/fonu/fonu/internal/certificate"
)

// stageCertFiles stages every PEM before replacing any live file.
// finish(false) restores previous files; finish(true) removes backups.
// This is process-level rollback, not a crash-atomic multi-file transaction.
func stageCertFiles(root string, domains []string, certPEM,keyPEM []byte) (string,string,func(bool)error,error) {
 if _,err:=tls.X509KeyPair(certPEM,keyPEM);err!=nil{return "","",nil,fmt.Errorf("证书与私钥校验失败：%w",err)}
 type item struct { dst,stage,backup string; replaced bool }
 var items []*item
 var dirs []string
 done:=false
 finish:=func(commit bool) error {
  if done{return nil};done=true
  var errs []error
  for i:=len(items)-1;i>=0;i-- { v:=items[i]
   if !commit && v.replaced {
    if v.backup!="" {if err:=os.Rename(v.backup,v.dst);err!=nil{errs=append(errs,err);continue};v.backup=""} else if err:=os.Remove(v.dst);err!=nil&&!os.IsNotExist(err){errs=append(errs,err)}
   }
   if v.stage!="" {_=os.Remove(v.stage)}
   if v.backup!="" {_=os.Remove(v.backup)}
  }
  if !commit {for i:=len(dirs)-1;i>=0;i--{_=os.Remove(dirs[i])}}
  return errors.Join(errs...)
 }
 fail:=func(err error)(string,string,func(bool)error,error){return "","",nil,errors.Join(err,finish(false))}
 if err:=os.MkdirAll(root,0755);err!=nil{return fail(err)}
 var cp,kp string
 for _,name:=range certstore.StorageNames(domains){
  if name==""||name=="."||name==".."||filepath.Base(name)!=name{return fail(fmt.Errorf("无效证书存储名称"))}
  dir:=filepath.Join(root,name)
  if st,err:=os.Lstat(dir);err==nil {if !st.IsDir()||st.Mode()&os.ModeSymlink!=0{return fail(fmt.Errorf("证书目录不安全"))}} else if os.IsNotExist(err){if err=os.Mkdir(dir,0755);err!=nil{return fail(err)};dirs=append(dirs,dir)}else{return fail(err)}
  cp=filepath.Join(dir,"fullchain.pem");kp=filepath.Join(dir,"privatekey.pem")
  for _,p:=range []struct{dst string; data []byte}{{cp,certPEM},{kp,keyPEM}} {
   v:= &item{dst:p.dst};items=append(items,v)
   f,err:=os.CreateTemp(dir,".fonu-stage-*");if err!=nil{return fail(err)};v.stage=f.Name()
   _,err=f.Write(p.data);if err==nil{err=f.Sync()};ce:=f.Close();if err==nil{err=ce};if err!=nil{return fail(err)}
   if st,err:=os.Lstat(p.dst);err==nil{
    if !st.Mode().IsRegular(){return fail(fmt.Errorf("证书文件不是普通文件"))}
    backup,err:=os.CreateTemp(dir,".fonu-backup-*");if err!=nil{return fail(err)};v.backup=backup.Name()
    old,err:=os.ReadFile(p.dst);if err==nil{_,err=backup.Write(old)};if err==nil{err=backup.Sync()};ce:=backup.Close();if err==nil{err=ce};if err!=nil{return fail(err)}
   }else if !os.IsNotExist(err){return fail(err)}
  }
 }
 if len(items)==0{return fail(fmt.Errorf("证书域名不能为空"))}
 for _,v:=range items{if err:=os.Rename(v.stage,v.dst);err!=nil{return fail(err)};v.stage="";v.replaced=true}
 return cp,kp,finish,nil
}
