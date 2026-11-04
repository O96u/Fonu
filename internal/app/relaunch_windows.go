//go:build windows

package app

import (
	"os"
	"os/exec"
)

func replaceProcess(exe string) error {
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
