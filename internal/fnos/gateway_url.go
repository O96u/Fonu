package fnos

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NormalizeGatewayURL checks an administrator-supplied fnOS entry.
// The host must be the browser's current host, or a loopback/private address.
// The path is always the gateway prefix. Query strings and credentials are rejected.
func NormalizeGatewayURL(raw, requestHost string) (string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", errors.New("网关地址无效")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("网关地址无效")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("网关地址无效")
	}
	host := parsed.Hostname()
	if !hostAllowed(host, requestHost) {
		return "", errors.New("网关地址只能是当前主机或内网地址")
	}
	if port := parsed.Port(); port != "" {
		if _, err := net.LookupPort("tcp", port); err != nil {
			return "", errors.New("网关地址无效")
		}
	}
	path := strings.TrimRight(parsed.Path, "/")
	if path != Prefix() {
		return "", errors.New("网关路径必须是飞牛里的 Fonu 入口")
	}
	parsed.Path = Prefix() + "/"
	parsed.RawPath = ""
	return parsed.String(), nil
}

// VerifyGateway confirms the address answers, without following a redirect
// to another host and without ignoring TLS errors.
func VerifyGateway(ctx context.Context, raw, requestHost string) (string, error) {
	normalized, err := NormalizeGatewayURL(raw, requestHost)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, normalized, nil)
	if err != nil {
		return "", errors.New("网关地址无效")
	}
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			Proxy:                 nil,
			TLSHandshakeTimeout:   3 * time.Second,
			ResponseHeaderTimeout: 3 * time.Second,
			DialContext:           (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		if certificateError(err) {
			return "", errors.New("飞牛网关证书无效，已拒绝连接")
		}
		return "", errors.New("无法连接飞牛认证网关，请检查 NAS 地址或网络连接")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return normalized, nil
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc, locErr := resp.Location()
		if locErr != nil || loc == nil || !strings.EqualFold(loc.Hostname(), req.URL.Hostname()) {
			return "", errors.New("飞牛网关跳转到了未允许的地址")
		}
		return normalized, nil
	}
	return "", fmt.Errorf("无法确认这是飞牛里的 Fonu 入口")
}

func hostAllowed(host, requestHost string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	requestHost = strings.Trim(strings.TrimSpace(requestHost), "[]")
	if host == "" {
		return false
	}
	if requestHost != "" && strings.EqualFold(host, requestHost) {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

func certificateError(err error) bool {
	var unknown x509.UnknownAuthorityError
	var invalid x509.CertificateInvalidError
	var host x509.HostnameError
	return errors.As(err, &unknown) || errors.As(err, &invalid) || errors.As(err, &host)
}

func sameHostname(raw, requestHost string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), strings.Trim(strings.TrimSpace(requestHost), "[]"))
}
