//go:build !linux

package nginx

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (m *Manager) ownsMaster(pid int) bool {
	if pid<=1||!isPIDAlive(pid){return false}
	ctx,cancel:=context.WithTimeout(context.Background(),2*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS=="windows"{
		script:=fmt.Sprintf("(Get-CimInstance Win32_Process -Filter 'ProcessId = %d').CommandLine",pid)
		cmd=exec.CommandContext(ctx,"powershell","-NoProfile","-NonInteractive","-Command",script)
	}else{
		cmd=exec.CommandContext(ctx,"ps","-p",strconv.Itoa(pid),"-o","command=")
	}
	out,err:=cmd.Output()
	if err!=nil{return false}
	raw:=strings.ToLower(strings.ReplaceAll(string(out),"\\","/"))
	configPath:=strings.ToLower(absNginxPath(m.cfg.NginxConfigPath()))
	return strings.Contains(raw,"nginx")&&strings.Contains(raw,configPath)
}
