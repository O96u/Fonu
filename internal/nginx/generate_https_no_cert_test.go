package nginx

import (
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateHTTPSWithoutCertFailsClosed(t *testing.T) {
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

	if _, err := Generate(cfg, rules, nil, GenerateOptions{}); err == nil {
		t.Fatal("expected missing certificate to reject HTTPS configuration")
	}
}
