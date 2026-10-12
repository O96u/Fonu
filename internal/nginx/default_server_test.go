package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateUnmatchedHostReturns404(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs", "wildcard.example.com")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "fullchain.pem"), []byte("cert"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "privatekey.pem"), []byte("key"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		DataDir:        dir,
		NginxPIDFile:   filepath.Join(dir, "nginx.pid"),
		NginxMimeTypes: filepath.Join(dir, "mime.types"),
	}
	rules := []proxy.Rule{{
		ListenPort:   8011,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "app.example.com"}},
		Upstream:     "http://127.0.0.1:8080",
		HTTPSEnabled: true,
		Enabled:      true,
	}}

	content, err := Generate(cfg, rules, []CertSource{{
		Domains:  []string{"*.example.com"},
		CertPath: filepath.Join(certDir, "fullchain.pem"),
		KeyPath:  filepath.Join(certDir, "privatekey.pem"),
	}}, GenerateOptions{})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if !strings.Contains(content, "listen 8011 ssl default_server;") {
		t.Fatalf("expected default_server catch-all on 8011:\n%s", content)
	}
	if !strings.Contains(content, "server_name _;") {
		t.Fatalf("expected catch-all server_name _:\n%s", content)
	}
	idxDefault := strings.Index(content, "server_name _;")
	idxReturn404 := strings.Index(content[idxDefault:], "return 404;")
	if idxReturn404 < 0 {
		t.Fatalf("expected return 404 in default server block:\n%s", content)
	}
}
