package nginx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateMixedRulesWithoutCerts(t *testing.T) {
	dir := t.TempDir()
	mimePath := filepath.Join(dir, "mime.types")
	if err := os.WriteFile(mimePath, []byte("types { text/html html; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		DataDir:        dir,
		NginxPIDFile:   filepath.Join(dir, "nginx.pid"),
		NginxMimeTypes: mimePath,
	}

	rules := []proxy.Rule{
		{
			ID: 1, Upstream: "http://192.168.1.10:5666", ListenPort: 443, ListenIPv4: true,
			Hosts: []proxy.Host{{Hostname: "nas.example.com"}},
			HTTPSEnabled: true, HTTPRedirect: true, Enabled: true,
		},
		{
			ID: 2, Upstream: "http://192.168.8.3:6893", ListenPort: 8011, ListenIPv4: true, ListenIPv6: true,
			Hosts: []proxy.Host{{Hostname: "app.example.com"}},
			HTTPSEnabled: true, HTTPRedirect: true, Enabled: true,
		},
	}

	content, err := Generate(cfg, rules, nil)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if err := assertValidSSLBlocks(content); err != nil {
		t.Fatalf("invalid ssl blocks: %v\n%s", err, content)
	}
}
