package certexport

import (
	"fmt"
	"path/filepath"
	"strings"
)

func cleanAbs(path string) string {
	return filepath.Clean(path)
}

// ValidateTargetPath ensures a mapping target is safe and distinct from the certs root.
func ValidateTargetPath(target, certsDir string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("目标目录不能为空")
	}
	if !filepath.IsAbs(target) {
		return fmt.Errorf("目标目录必须为绝对路径：%s", target)
	}
	target = cleanAbs(target)
	certsDir = cleanAbs(certsDir)
	if target == certsDir {
		return fmt.Errorf("目标目录不能与证书源目录相同")
	}
	if isUnderDir(target, certsDir) {
		return fmt.Errorf("目标目录不能位于证书源目录内：%s", target)
	}
	return nil
}

func isUnderDir(path, parent string) bool {
	path = cleanAbs(path)
	parent = cleanAbs(parent)
	if path == parent {
		return true
	}
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
