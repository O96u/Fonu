package notify

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeProxyURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("Telegram 代理地址无效")
	}
	switch parsed.Scheme {
	case "http", "https", "socks5", "socks5h":
	default:
		return "", fmt.Errorf("Telegram 代理仅支持 http、https、socks5")
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return "", fmt.Errorf("Telegram 代理地址无效")
	}
	if parsed.Port() == "" {
		return "", fmt.Errorf("Telegram 代理地址需包含端口，例如 http://127.0.0.1:7890")
	}
	return parsed.String(), nil
}
