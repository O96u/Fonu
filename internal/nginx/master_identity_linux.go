//go:build linux

package nginx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (m *Manager) ownsMaster(pid int) bool {
	if pid <= 1 || !isPIDAlive(pid) { return false }
	bin,err:=exec.LookPath(m.cfg.NginxBin)
	if err!=nil{return false}
	expected,err:=os.Stat(bin)
	if err!=nil{return false}
	actual,err:=os.Stat(fmt.Sprintf("/proc/%d/exe",pid))
	if err!=nil||!os.SameFile(expected,actual){return false}
	raw,err:=os.ReadFile(fmt.Sprintf("/proc/%d/cmdline",pid))
	if err!=nil{return false}
	cmd:=strings.TrimSpace(strings.ReplaceAll(string(raw),"\x00"," "))
	configPath:=filepath.ToSlash(absNginxPath(m.cfg.NginxConfigPath()))
	return strings.HasPrefix(cmd,"nginx: master process ") && strings.Contains(cmd+" "," -c "+configPath+" ")
}
