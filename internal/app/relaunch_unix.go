//go:build !windows

package app

import (
	"os"
	"syscall"
)

func replaceProcess(exe string) error {
	args := append([]string{exe}, os.Args[1:]...)
	return syscall.Exec(exe, args, os.Environ())
}
