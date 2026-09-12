package nginx

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestApplyWithNginxIfAvailable(t *testing.T) {
	if _, err := exec.LookPath("nginx"); err != nil {
		t.Skip("nginx not installed")
	}

	dir := t.TempDir()
	mimePath := filepath.Join(dir, "mime.types")
	if err := os.WriteFile(mimePath, []byte("types { text/html html; }\n"), 0o644); err != nil {
		t.Fatalf("write mime.types: %v", err)
	}

	cfg := config.Config{
		DataDir:        dir,
		NginxBin:       "nginx",
		NginxPIDFile:   filepath.Join(dir, "nginx", "nginx.pid"),
		NginxMimeTypes: mimePath,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mgr := NewManager(cfg, logger)

	rules := []proxy.Rule{{
		ID:           1,
		Upstream:     "http://127.0.0.1:65535",
		ListenPort:   18080,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "nas.example.com"}},
		HTTPSEnabled: false,
		HTTPRedirect: false,
		Enabled:      true,
	}}

	if _, err := mgr.Apply(context.Background(), rules, nil); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	defer mgr.Stop(context.Background())

	tmp := filepath.Join(dir, "nginx", "bad.conf")
	if err := os.WriteFile(tmp, []byte("this is not valid nginx config"), 0o644); err != nil {
		t.Fatalf("write bad config: %v", err)
	}
	if err := mgr.validate(context.Background(), tmp); err == nil {
		t.Fatal("expected invalid config to fail validation")
	}
}
