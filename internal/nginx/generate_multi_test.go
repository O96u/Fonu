package nginx

import (
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateMultiHostCustomPort(t *testing.T) {
	cfg := config.Config{
		DataDir:        t.TempDir(),
		NginxPIDFile:   t.TempDir() + "/nginx.pid",
		NginxMimeTypes: t.TempDir() + "/mime.types",
	}
	port6893 := 6893
	rules := []proxy.Rule{{
		ID:           1,
		Upstream:     "http://192.168.8.3:6893",
		ListenPort:   8011,
		ListenIPv4:   true,
		ListenIPv6:   true,
		Hosts: []proxy.Host{
			{Hostname: "1.example.com"},
			{Hostname: "2.example.com"},
			{Hostname: "example.org"},
			{Hostname: "example.com", ListenPort: &port6893},
		},
		HTTPSEnabled: false,
		HTTPRedirect: false,
		Enabled:      true,
	}}

	content, err := Generate(cfg, rules, nil, GenerateOptions{})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	for _, want := range []string{
		"listen 8011;",
		"listen [::]:8011;",
		"server_name 1.example.com 2.example.com example.org",
		"listen 6893;",
		"listen [::]:6893;",
		"server_name example.com",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in generated config:\n%s", want, content)
		}
	}
}
