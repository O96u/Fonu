package ddns

import (
	"context"
	"net/http"
	"strings"
)

// RegisterResult 是新域名注册结果。
type RegisterResult struct {
	SubdomainID int64  `json:"subdomain_id"`
	FullDomain  string `json:"full_domain"`
}

// RegisterSubdomain 注册新的免费子域名。
// prefix 为子域名前缀（仅小写字母/数字/连字符，免费前缀长度需 ≥4 位），
// rootDomain 为 DNSHE 允许的主域名后缀；前缀过短等情况 DNSHE 返回 402。
func (d *DNSHE) RegisterSubdomain(ctx context.Context, cred Credentials, prefix, rootDomain string) (*RegisterResult, error) {
	payload := map[string]any{
		"subdomain": strings.ToLower(strings.TrimSpace(prefix)),
		"domain":    strings.ToLower(strings.TrimSpace(rootDomain)),
	}
	var resp struct {
		dnsheBase
		SubdomainID int64  `json:"subdomain_id"`
		FullDomain  string `json:"full_domain"`
	}
	if err := d.call(ctx, cred, http.MethodPost, "subdomains", "create", nil, payload, &resp); err != nil {
		return nil, err
	}
	d.cacheMu.Lock()
	delete(d.cache, cred.Token)
	d.cacheMu.Unlock()
	return &RegisterResult{SubdomainID: resp.SubdomainID, FullDomain: resp.FullDomain}, nil
}
