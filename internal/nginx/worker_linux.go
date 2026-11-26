//go:build linux

package nginx

import (
	"fmt"
	"os"
	"strings"
)

func workerConfirmationSupported()bool{return true}
func activeWorkers(pid int)map[string]bool{
	out:=map[string]bool{}
	children,err:=os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children",pid,pid))
	if err!=nil{return out}
	for _,child:=range strings.Fields(string(children)){
		cmd,err:=os.ReadFile("/proc/"+child+"/cmdline")
		if err!=nil||strings.TrimSpace(strings.ReplaceAll(string(cmd),"\x00"," "))!="nginx: worker process"{continue}
		stat,err:=os.ReadFile("/proc/"+child+"/stat")
		if err!=nil{continue}
		end:=strings.LastIndex(string(stat),")")
		if end<0{continue}
		fields:=strings.Fields(string(stat)[end+1:])
		if len(fields)<20||fields[0]=="Z"{continue}
		out[child+":"+fields[19]]=true
	}
	return out
}
