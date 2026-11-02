package nginx

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BinAvailable reports whether the configured nginx executable can be invoked.
func BinAvailable(bin string) bool {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		return false
	}
	if strings.Contains(bin, string(os.PathSeparator)) || filepath.IsAbs(bin) {
		info, err := os.Stat(bin)
		return err == nil && !info.IsDir()
	}
	_, err := exec.LookPath(bin)
	return err == nil
}
