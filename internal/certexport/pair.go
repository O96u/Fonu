package certexport

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// pairOps permits the same rollback logic for local disk and SFTP.
type pairOps struct{
	stat func(string)(os.FileInfo,error)
	rename func(string,string)error
	remove func(string)error
	write func(string,[]byte)error
}

type pemItem struct{
	dst,stage,backup string
	data []byte
	existing,staged,saved,published bool
}

// publishPEMPair stages both files before touching the destination. A failure
// restores the old files in reverse order. Recovery files are retained if a
// rollback fails. This is not a crash-atomic multi-file swap.
func publishPEMPair(srcDir,dstDir string,join func(...string)string,ops pairOps)(err error){
	id:=uuid.NewString()
	items:=make([]pemItem,0,2)
	for _,name:=range []string{certFileName,keyFileName}{
		src:=filepath.Join(srcDir,name)
		info,e:=os.Lstat(src);if e!=nil{return e}
		if !info.Mode().IsRegular(){return fmt.Errorf("证书源文件不是普通文件")}
		data,e:=os.ReadFile(src);if e!=nil{return e}
		item:=pemItem{dst:join(dstDir,name),stage:join(dstDir,".fonu-stage-"+id+"-"+name),backup:join(dstDir,".fonu-backup-"+id+"-"+name),data:data}
		if info,e:=ops.stat(item.dst);e==nil{
			if !info.Mode().IsRegular(){return fmt.Errorf("证书目标不是普通文件")}
			item.existing=true
		}else if !os.IsNotExist(e){return e}
		items=append(items,item)
	}
	remove:=func(name string)error{e:=ops.remove(name);if os.IsNotExist(e){return nil};return e}
	defer func(){
		committed:=err==nil
		for i:=len(items)-1;i>=0;i--{
			v:=&items[i]
			if !committed{
				if v.published {
					if e:=remove(v.dst);e!=nil{
						err=errors.Join(err,fmt.Errorf("撤回新证书失败：%w",e))
						continue
					}
				}
				if v.saved {
					if e:=ops.rename(v.backup,v.dst);e!=nil{
						err=errors.Join(err,fmt.Errorf("恢复旧证书失败，已保留 %s：%w",v.backup,e))
						continue
					}
					v.saved=false
				}
			}
			if v.staged{err=errors.Join(err,remove(v.stage))}
			if v.saved&&committed{err=errors.Join(err,remove(v.backup))}
		}
	}()
	for i:=range items{
		v:=&items[i]
		if e:=ops.write(v.stage,v.data);e!=nil{return fmt.Errorf("暂存证书失败：%w",e)}
		v.staged=true
	}
	for i:=range items{
		v:=&items[i]
		if v.existing{
			if e:=ops.rename(v.dst,v.backup);e!=nil{return fmt.Errorf("保存旧证书失败：%w",e)}
			v.saved=true
		}
		if e:=ops.rename(v.stage,v.dst);e!=nil{return fmt.Errorf("发布证书失败：%w",e)}
		v.staged=false;v.published=true
	}
	return nil
}
