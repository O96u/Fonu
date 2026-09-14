package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestChinaOnlyBypassGeoWithoutUserWhitelist(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mime.types"), []byte("types { text/html html; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cidrPath := filepath.Join(dir, "nginx", "china_cidr.conf")
	if err := os.MkdirAll(filepath.Dir(cidrPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cidrPath, []byte("1.0.0.0/8 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		DataDir:        dir,
		NginxPIDFile:   filepath.Join(dir, "nginx.pid"),
		NginxMimeTypes: filepath.Join(dir, "mime.types"),
	}
	rules := []proxy.Rule{{
		ID:           3,
		ListenPort:   8080,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "app.example.com"}},
		Upstream:     "http://127.0.0.1:3000",
		Enabled:      true,
		HTTPSEnabled: false,
		Security:     proxy.SecurityConfig{ChinaOnly: true},
	}}

	content, err := Generate(cfg, rules, nil, GenerateOptions{ChinaCIDRAvailable: true})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		"geo $fonu_client_ip $fonu_rule_3_china_bypass",
		"192.168.0.0/16 1;",
		"10.0.0.0/8 1;",
		"if ($fonu_rule_3_china_bypass = 1)",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in:\n%s", want, content)
		}
	}
}
