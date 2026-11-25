package nginx

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fonu/fonu/internal/config"
)

type PublicationFiles struct{
	Main []byte `json:"main"`
	Global []byte `json:"global"`
	Rules map[string][]byte `json:"rules"`
	Auth map[string][]byte `json:"auth"`
}

func readOptionalRegular(path string)([]byte,error){
	info,err:=os.Lstat(path)
	if os.IsNotExist(err){return nil,nil}
	if err!=nil{return nil,err}
	if !info.Mode().IsRegular()||info.Size()>16<<20{return nil,fmt.Errorf("invalid managed config file: %s",path)}
	data,err:=os.ReadFile(path)
	if data==nil&&err==nil{data=[]byte{}}
	return data,err
}

func SnapshotPublicationFiles(cfg config.Config)(PublicationFiles,error){
	var out PublicationFiles
	var err error
	if out.Main,err=readOptionalRegular(cfg.NginxConfigPath());err!=nil{return out,err}
	if out.Global,err=readOptionalRegular(cfg.NginxGlobalCustomPath());err!=nil{return out,err}
	out.Rules=map[string][]byte{}
	entries,err:=os.ReadDir(cfg.NginxRulesDir())
	if err!=nil&&!os.IsNotExist(err){return out,err}
	for _,entry:=range entries{
		if !managedHtpasswd(entry.Name()){continue}
		data,err:=readOptionalRegular(filepath.Join(cfg.NginxRulesDir(),entry.Name()))
		if err!=nil{return out,err}
		out.Rules[entry.Name()]=data
	}
	out.Auth,err=snapshotHtpasswd(cfg)
	return out,err
}

func restoreOptional(path string,data []byte)error{
	if data!=nil{return WriteCustomAtomic(path,string(data))}
	err:=os.Remove(path)
	if os.IsNotExist(err){return nil}
	return err
}

func RestorePublicationFiles(cfg config.Config,files PublicationFiles)error{
	if files.Rules==nil||files.Auth==nil{return fmt.Errorf("incomplete publication snapshot")}
	for name:=range files.Rules{if !managedHtpasswd(name){return fmt.Errorf("invalid rule snapshot name")}}
	for name:=range files.Auth{if !managedHtpasswd(name){return fmt.Errorf("invalid auth snapshot name")}}
	if err:=restoreOptional(cfg.NginxGlobalCustomPath(),files.Global);err!=nil{return err}
	entries,err:=os.ReadDir(cfg.NginxRulesDir())
	if err!=nil&&!os.IsNotExist(err){return err}
	for _,entry:=range entries{
		name:=entry.Name()
		if !managedHtpasswd(name){continue}
		if _,ok:=files.Rules[name];!ok{
			if err:=restoreOptional(filepath.Join(cfg.NginxRulesDir(),name),nil);err!=nil{return err}
		}
	}
	for name,data:=range files.Rules{
		if err:=restoreOptional(filepath.Join(cfg.NginxRulesDir(),name),data);err!=nil{return err}
	}
	if err:=restoreHtpasswd(cfg,files.Auth);err!=nil{return err}
	return restoreOptional(cfg.NginxConfigPath(),files.Main)
}
