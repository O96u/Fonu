//go:build !linux && !darwin && !freebsd && !windows

package nginx

import (
	"fmt"
	"os"
	"syscall"
)

func reloadProcess(pid int) error {
	return fmt.Errorf("signal reload unsupported")
}

func terminateProcess(pid int, sig syscall.Signal) error {
	return fmt.Errorf("signal terminate unsupported")
}

func isPIDAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
