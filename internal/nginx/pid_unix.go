//go:build linux || darwin || freebsd

package nginx

import (
	"os"
	"syscall"
)

func reloadProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGHUP)
}

func terminateProcess(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}

func isPIDAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
