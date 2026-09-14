package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateChinaCIDRUsesFinalPathWhenOverrideCleared(t *testing.T) {
	dir := t.TempDir()
	cidrDir := filepath.Join(dir, "nginx")
	if err := os.MkdirAll(cidrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	finalPath := filepath.Join(cidrDir, "china_cidr.conf")
	if err := os.WriteFile(finalPath, []byte("1.0.0.0/8 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		DataDir:        dir,
		NginxPIDFile:   filepath.Join(dir, "nginx.pid"),
		NginxMimeTypes: filepath.Join(dir, "mime.types"),
	}
	if err := os.WriteFile(cfg.NginxMimeTypes, []byte("types { text/html html; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rules := []proxy.Rule{{
		ID:         1,
		ListenPort: 8080,
		ListenIPv4: true,
		Hosts:      []proxy.Host{{Hostname: "app.example.com"}},
		Upstream:   "http://127.0.0.1:3000",
		Enabled:    true,
		Security:   proxy.SecurityConfig{ChinaOnly: true},
	}}

	opts := GenerateOptions{
		ChinaCIDRAvailable:    true,
		ChinaCIDRPathOverride: filepath.Join(cidrDir, "china_cidr.conf.tmp"),
	}
	opts.ChinaCIDRPathOverride = ""

	content, err := Generate(cfg, rules, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "china_cidr.conf.tmp") {
		t.Fatalf("expected final china cidr path, got tmp reference:\n%s", content)
	}
	if !strings.Contains(content, "china_cidr.conf") {
		t.Fatalf("missing china_cidr.conf include:\n%s", content)
	}
}
