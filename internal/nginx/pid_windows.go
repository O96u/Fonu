//go:build windows

package nginx

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

const stillActive = 259

func reloadProcess(pid int) error {
	return fmt.Errorf("signal reload unsupported on windows")
}

func terminateProcess(pid int, sig syscall.Signal) error {
	args := []string{"/PID", strconv.Itoa(pid)}
	if sig == syscall.SIGKILL {
		args = append([]string{"/F"}, args...)
	}
	cmd := exec.Command("taskkill", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("taskkill: %w", err)
	}
	return nil
}

func isPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == stillActive
}

func stopOrphanMasters(nginxDir string) {
	marker := strings.ToLower(filepath.ToSlash(absNginxPath(nginxDir)))
	marker = strings.ReplaceAll(marker, "'", "''")
	script := fmt.Sprintf(
		`Get-CimInstance Win32_Process -Filter "Name='nginx.exe'" | Where-Object { $_.CommandLine -like '*%s*' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`,
		marker,
	)
	_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Run()
}
