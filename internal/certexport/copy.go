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

func removePEMPair(dir string) error {
	for _, name := range []string{certFileName, keyFileName} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) { return err }
	}
	return nil
}

func copyFileAtomic(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(dst), ".fonu-*")
	if err != nil { return err }
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err == nil { _, err = f.Write(data) }
	if err == nil { err = f.Sync() }
	closeErr := f.Close()
	if err != nil { return err }
	if closeErr != nil { return closeErr }
	return os.Rename(tmp, dst)
}
