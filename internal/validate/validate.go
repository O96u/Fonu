package validate

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var domainRe = regexp.MustCompile(`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)(?:\.(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?))*$`)

func Domain(domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("域名不能为空")
	}
	if len(domain) > 253 {
		return fmt.Errorf("域名过长")
	}
	if strings.Contains(domain, "..") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("域名格式无效")
	}
	if !domainRe.MatchString(domain) {
		return fmt.Errorf("域名格式无效")
	}
	return nil
}

func Upstream(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("目标地址不能为空")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("目标地址格式无效")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("目标地址仅支持 http:// 或 https://")
	}
	if u.User != nil {
		return "", fmt.Errorf("目标地址不能包含用户名或密码")
	}
	if u.Host == "" {
		return "", fmt.Errorf("目标地址缺少主机名")
	}
	if u.Path != "" && u.Path != "/" {
		return "", fmt.Errorf("目标地址不能包含路径")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("目标地址不能包含查询参数或片段")
	}

	host := u.Hostname()
	port := u.Port()
	if host == "" {
		return "", fmt.Errorf("目标地址主机名无效")
	}
	if ip := net.ParseIP(host); ip != nil {
		if port == "" {
			return "", fmt.Errorf("IP 地址目标必须指定端口")
		}
	} else if err := Domain(host); err != nil {
		return "", fmt.Errorf("目标地址主机名无效")
	}

	normalized := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	if !strings.HasPrefix(normalized, "http://") && !strings.HasPrefix(normalized, "https://") {
		return "", fmt.Errorf("目标地址格式无效")
	}
	return normalized, nil
}

func Password(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("密码长度至少 8 位")
	}
	return nil
}

func Username(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(username) > 64 {
		return fmt.Errorf("用户名过长")
	}
	return nil
}
