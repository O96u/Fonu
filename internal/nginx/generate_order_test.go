package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateHTTPSBlockBeforeHTTPRedirect(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(dir, "mime.types"), []byte("types { text/html html; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		DataDir:        dir,
		NginxPIDFile:   filepath.Join(dir, "nginx", "nginx.pid"),
		NginxMimeTypes: filepath.Join(dir, "mime.types"),
	}
	rules := []proxy.Rule{{
		ListenPort:   8017,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "app.example.com"}},
		Upstream:     "http://127.0.0.1:1",
		HTTPSEnabled: true,
		HTTPRedirect: true,
		Enabled:      true,
	}}
	certs := []CertSource{{
		Domains:  []string{"*.example.com"},
		CertPath: filepath.Join(certDir, "fullchain.pem"),
		KeyPath:  filepath.Join(certDir, "privatekey.pem"),
	}}

	content, err := Generate(cfg, rules, certs, GenerateOptions{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	sslIdx := strings.Index(content, "listen 8017 ssl;")
	redirectIdx := strings.Index(content, "return 301 https://")
	if sslIdx < 0 || redirectIdx < 0 {
		t.Fatalf("missing expected directives:\n%s", content)
	}
	if sslIdx > redirectIdx {
		t.Fatalf("expected ssl server block before http redirect:\n%s", content)
	}
}
