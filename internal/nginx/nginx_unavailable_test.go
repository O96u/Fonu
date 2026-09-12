package nginx

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestApplyWithoutNginxBinary(t *testing.T) {
	dir := t.TempDir()
	mimePath := filepath.Join(dir, "mime.types")
	if err := os.WriteFile(mimePath, []byte("types { text/html html; }\n"), 0o644); err != nil {
		t.Fatalf("write mime.types: %v", err)
	}

	cfg := config.Config{
		DataDir:        dir,
		NginxBin:       "fonu-missing-nginx-binary",
		NginxPIDFile:   filepath.Join(dir, "nginx", "nginx.pid"),
		NginxMimeTypes: mimePath,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mgr := NewManager(cfg, logger)

	rules := []proxy.Rule{{
		ID:           1,
		Upstream:     "http://127.0.0.1:65535",
		ListenPort:   8011,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "nas.example.com"}},
		HTTPSEnabled: false,
		Enabled:      true,
	}}

	result, err := mgr.Apply(context.Background(), rules)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Message == "" {
		t.Fatal("expected skip message")
	}
	if _, err := os.Stat(cfg.NginxConfigPath()); err != nil {
		t.Fatalf("config not written: %v", err)
	}
}
