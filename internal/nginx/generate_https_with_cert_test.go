package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateHTTPSWithCertIncludesCertificateDirectives(t *testing.T) {
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
		ListenIPv6:   true,
		Hosts:        []proxy.Host{{Hostname: "app.example.com"}},
		Upstream:     "http://192.168.8.3:6893",
		HTTPSEnabled: true,
		HTTPRedirect: true,
		Enabled:      true,
	}}

	content, err := Generate(cfg, rules, []CertSource{{
		Domains:  []string{"*.example.com", "example.com"},
		CertPath: filepath.Join(certDir, "fullchain.pem"),
		KeyPath:  filepath.Join(certDir, "privatekey.pem"),
	}}, GenerateOptions{})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if !strings.Contains(content, "listen 8011 ssl;") {
		t.Fatalf("expected ssl listen:\n%s", content)
	}
	if !strings.Contains(content, "ssl_certificate") || !strings.Contains(content, "ssl_certificate_key") {
		t.Fatalf("expected certificate directives:\n%s", content)
	}
}
