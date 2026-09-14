package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateSecurityDirectives(t *testing.T) {
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
		ID:           7,
		ListenPort:   8080,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "app.example.com"}},
		Upstream:     "http://127.0.0.1:3000",
		Enabled:      true,
		HTTPSEnabled: false,
		Security: proxy.SecurityConfig{
			IPBlacklist:     []string{"1.2.3.4", "10.0.0.0/8"},
			IPWhitelistMode: true,
			IPWhitelist:     []string{"192.168.1.0/24"},
			ChinaOnly:       true,
			BasicAuth: &proxy.BasicAuthConfig{
				Enabled:      true,
				Username:     "user",
				PasswordHash: "$2a$10$abcdefghijklmnopqrstuv",
			},
			RateLimit: &proxy.RateLimitConfig{Enabled: true, Rate: 10, Burst: 20},
			ConnLimit: &proxy.ConnLimitConfig{Enabled: true, Max: 20},
		},
	}}

	opts := GenerateOptions{
		GlobalIPBlacklist:  []string{"8.8.8.8"},
		ChinaCIDRAvailable: true,
	}
	content, err := Generate(cfg, rules, nil, opts)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	checks := []string{
		"deny 8.8.8.8;",
		"limit_req_zone $binary_remote_addr zone=fonu_rule_7_req",
		"limit_conn_zone $binary_remote_addr zone=fonu_rule_7_conn",
		"deny 1.2.3.4;",
		"geo $fonu_client_ip $fonu_is_private",
		"geo $fonu_client_ip $fonu_rule_7_china_bypass",
		"192.168.0.0/16 1;",
		"192.168.1.0/24 1;",
		"set $fonu_china_block 0;",
		"if ($fonu_is_private = 1)",
		"if ($fonu_rule_7_china_bypass = 1)",
		"allow 192.168.1.0/24;",
		"deny all;",
		"auth_basic",
		"limit_req zone=fonu_rule_7_req burst=20",
		"limit_conn fonu_rule_7_conn 20",
		"error_page 403 /fonu-errors/403.html",
		"location = /fonu-errors/error.png",
		"location = /fonu-errors/429.png",
		"X-Forwarded-Host $host",
		"X-Forwarded-Port $server_port",
		"X-Real-Proto $scheme",
	}
	for _, want := range checks {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in:\n%s", want, content)
		}
	}
}
