//go:build linux || darwin || freebsd

package nginx

import "syscall"

func reloadProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGHUP)
}

func terminateProcess(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}
