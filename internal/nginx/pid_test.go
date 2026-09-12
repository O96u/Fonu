package nginx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/config"
)

func TestReadPIDFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nginx.pid")

	if _, err := readPIDFile(path); !os.IsNotExist(err) {
		t.Fatalf("expected not exist, got %v", err)
	}

	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPIDFile(path); err == nil {
		t.Fatal("expected error for empty pid file")
	}

	if err := os.WriteFile(path, []byte("12345\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pid, err := readPIDFile(path)
	if err != nil || pid != 12345 {
		t.Fatalf("expected pid 12345, got %d err=%v", pid, err)
	}
}

func TestIsRunningStalePIDFile(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "nginx.pid")
	if err := os.WriteFile(pidPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr := &Manager{cfg: config.Config{NginxPIDFile: pidPath}}
	running, err := mgr.isRunning()
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Fatal("expected stale empty pid file to be treated as not running")
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatal("expected stale pid file to be removed")
	}
}
