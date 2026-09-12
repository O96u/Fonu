//go:build !linux && !darwin && !freebsd

package nginx

import (
	"fmt"
	"syscall"
)

func reloadProcess(pid int) error {
	return fmt.Errorf("signal reload unsupported")
}

func terminateProcess(pid int, sig syscall.Signal) error {
	return fmt.Errorf("signal terminate unsupported")
}
