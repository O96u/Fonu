package publicip

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var ipv4Providers = []string{
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
	"https://ifconfig.me/ip",
	"http://ipv4.icanhazip.com",
}

var ipv6Providers = []string{
	"https://api64.ipify.org",
	"https://ipv6.icanhazip.com",
}

func Detect(ctx context.Context) (ipv4, ipv6 string, err error) {
	ipv4, v4err := detectIPv4(ctx)
	ipv6, _ = detectIPv6(ctx)
	if v4err != nil {
		return "", ipv6, v4err
	}
	return ipv4, ipv6, nil
}

func detectIPv4(ctx context.Context) (string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	var lastErr error
	for _, url := range ipv4Providers {
		ip, err := fetchIP(ctx, client, url, false)
		if err == nil {
			return ip, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("获取公网 IPv4 失败：%w", lastErr)
}

func detectIPv6(ctx context.Context) (string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	for _, url := range ipv6Providers {
		ip, err := fetchIP(ctx, client, url, true)
		if err == nil {
			return ip, nil
		}
	}
	return "", fmt.Errorf("no ipv6")
}

func fetchIP(ctx context.Context, client *http.Client, url string, wantV6 bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("invalid ip response")
	}
	if wantV6 && parsed.To4() != nil {
		return "", fmt.Errorf("no ipv6")
	}
	if !wantV6 && parsed.To4() == nil {
		return "", fmt.Errorf("no ipv4")
	}
	return ip, nil
}
