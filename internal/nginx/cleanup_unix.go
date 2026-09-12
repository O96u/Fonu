//go:build !windows

package nginx

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

func prepareStartPlatform(m *Manager, ctx context.Context) {
	stopOrphanMasters(ctx, m)
}

func stopOrphanMasters(ctx context.Context, m *Manager) {
	pid, err := readPIDFile(m.cfg.NginxPIDFile)
	if err == nil && isPIDAlive(pid) {
		_ = terminateProcess(pid, syscall.SIGTERM)
		waitProcessExit(pid, nginxTerminateWait)
		if isPIDAlive(pid) {
			_ = terminateProcess(pid, syscall.SIGKILL)
			waitProcessExit(pid, time.Second)
		}
	}
	_ = m.runNginx(ctx, []string{"-c", m.cfg.NginxConfigPath(), "-s", "quit"}, "")
	removePIDFile(m.cfg.NginxPIDFile)
	if _, err := exec.LookPath("pkill"); err == nil {
		_ = exec.Command("pkill", "-x", "nginx").Run()
		time.Sleep(200 * time.Millisecond)
	}
}
