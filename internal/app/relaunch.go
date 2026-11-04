package app

import (
	"os"
)

func relaunch() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return replaceProcess(exe)
}
