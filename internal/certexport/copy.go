package certexport

import (
	"fmt"
	"os"
	"path/filepath"
)

func copyPEMPair(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	if err := copyFileAtomic(filepath.Join(srcDir, certFileName), filepath.Join(dstDir, certFileName), 0o600); err != nil {
		return fmt.Errorf("复制 %s: %w", certFileName, err)
	}
	if err := copyFileAtomic(filepath.Join(srcDir, keyFileName), filepath.Join(dstDir, keyFileName), 0o600); err != nil {
		return fmt.Errorf("复制 %s: %w", keyFileName, err)
	}
	return nil
}

func copyFileAtomic(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dst + ".fonu.tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
