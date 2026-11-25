package nginx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreConfig(t *testing.T) {
	dir := t.TempDir()
	current, backup := filepath.Join(dir, "nginx.conf"), filepath.Join(dir, "nginx.conf.bak")
	if err := os.WriteFile(current, []byte("new"), 0600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(backup, []byte("old"), 0600); err != nil { t.Fatal(err) }
	if err := restoreConfig(current, backup, true); err != nil { t.Fatal(err) }
	got, err := os.ReadFile(current)
	if err != nil || string(got) != "old" { t.Fatalf("restored config: %q %v", got, err) }
	if err := restoreConfig(current, backup, false); err != nil { t.Fatal(err) }
	if _, err := os.Stat(current); !os.IsNotExist(err) { t.Fatalf("unexpected config: %v", err) }
}
