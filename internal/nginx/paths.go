package nginx

import "path/filepath"

func absNginxPath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(abs)
}
