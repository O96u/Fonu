package nginx

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(data))
	if pidStr == "" {
		return 0, fmt.Errorf("empty pid file")
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid pid: %q", pidStr)
	}
	return pid, nil
}

func removePIDFile(path string) {
	_ = os.Remove(path)
}

func waitProcessExit(pid int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isPIDAlive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
