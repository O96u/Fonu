package certexport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyAndRemovePEMPair(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "example.com")
	dstRoot := filepath.Join(root, "dst")
	_ = os.MkdirAll(src, 0o755)
	_ = os.WriteFile(filepath.Join(src, certFileName), []byte("cert"), 0o600)
	_ = os.WriteFile(filepath.Join(src, keyFileName), []byte("key"), 0o600)

	if err := copyPEMPair(src, filepath.Join(dstRoot, "example.com")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dstRoot, "example.com", certFileName)); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(filepath.Join(dstRoot, "example.com"))
}
