package nginx

import (
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateIncludesWebSocketAndProxy(t *testing.T) {
	cfg := config.Config{
		DataDir:        t.TempDir(),
		NginxPIDFile:   t.TempDir() + "/nginx.pid",
		NginxMimeTypes: t.TempDir() + "/mime.types",
	}
	rules := []proxy.Rule{{
		ID:           1,
		Upstream:     "http://192.168.1.10:5666",
		ListenPort:   80,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "nas.example.com"}},
		HTTPSEnabled: false,
		HTTPRedirect: false,
		Enabled:      true,
	}}

	content, err := Generate(cfg, rules, nil)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	for _, want := range []string{
		"listen 80;",
		"server_name nas.example.com",
		"proxy_pass http://192.168.1.10:5666",
		"proxy_set_header Upgrade $http_upgrade",
		"proxy_buffering off",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in generated config", want)
		}
	}
}
