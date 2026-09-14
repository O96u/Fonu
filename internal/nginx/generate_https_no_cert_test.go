package nginx

import (
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateHTTPSWithoutCertUsesHTTPOnly(t *testing.T) {
	cfg := config.Config{
		DataDir:        t.TempDir(),
		NginxPIDFile:   t.TempDir() + "/nginx.pid",
		NginxMimeTypes: t.TempDir() + "/mime.types",
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

	content, err := Generate(cfg, rules, nil, GenerateOptions{})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if strings.Contains(content, " ssl") {
		t.Fatalf("expected no ssl listen without certificate:\n%s", content)
	}
	if !strings.Contains(content, "proxy_pass http://192.168.8.3:6893") {
		t.Fatalf("expected http proxy fallback:\n%s", content)
	}
}
