package nginx

import (
	"os"
	"runtime"
)

// Debian/Ubuntu nginx worker user.
const nginxWorkerUID = 33
const nginxWorkerGID = 33

func ensureNginxReadable(path string, dir bool) {
	mode := os.FileMode(0o644)
	if dir {
		mode = 0o755
	}
	_ = os.Chmod(path, mode)
	if runtime.GOOS == "linux" {
		_ = os.Chown(path, nginxWorkerUID, nginxWorkerGID)
	}
}
