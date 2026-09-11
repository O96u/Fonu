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

const (
	ipv4URL = "https://api.ipify.org"
	ipv6URL = "https://api64.ipify.org"
)

func Detect(ctx context.Context) (ipv4, ipv6 string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	ipv4, err = fetchIP(ctx, client, ipv4URL, false)
	if err != nil {
		return "", "", fmt.Errorf("获取公网 IPv4 失败：%w", err)
	}
	ipv6, _ = fetchIP(ctx, client, ipv6URL, true)
	return ipv4, ipv6, nil
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
