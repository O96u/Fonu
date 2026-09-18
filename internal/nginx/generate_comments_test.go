package nginx

import (
	"strings"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
)

func TestGenerateRuleBlocksIncludesComments(t *testing.T) {
	cfg := config.Config{DataDir: t.TempDir()}
	rule := proxy.Rule{
		ID:           3,
		Name:         "fonu",
		Upstream:     "http://127.0.0.1:8080",
		ListenPort:   8011,
		ListenIPv4:   true,
		Hosts:        []proxy.Host{{Hostname: "s.example.com"}},
		HTTPSEnabled: false,
		Enabled:      true,
		Security: proxy.SecurityConfig{
			IPBlacklist:     []string{"192.168.8.2"},
			IPWhitelistMode: true,
			IPWhitelist:     []string{"192.168.8.1"},
			RateLimit:       &proxy.RateLimitConfig{Enabled: true, Rate: 10, Burst: 20},
			ConnLimit:       &proxy.ConnLimitConfig{Enabled: true, Max: 20},
		},
	}
	out, err := GenerateRuleBlocks(cfg, rule, nil, GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# 规则：fonu",
		"# 上游：http://127.0.0.1:8080",
		"# 限流 zone 在 http 块中自动生成",
		"# IP 黑名单",
		"# IP 白名单模式",
		"# 请求限流：10 次/秒，突发 20",
		"# 连接数限制：每 IP 最多 20 并发",
		"# 反代 location /",
		"# 反向代理与 WebSocket 头",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing comment %q in:\n%s", want, out)
		}
	}
}
