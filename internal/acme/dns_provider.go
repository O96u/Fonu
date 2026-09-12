package acme

import (
	"fmt"
	"strings"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/dnspod"

	"github.com/fonu/fonu/internal/ddns"
)

func newDNS01Provider(provider string, cred ddns.Credentials) (challenge.Provider, error) {
	switch provider {
	case "dnspod":
		cfg := dnspod.NewDefaultConfig()
		cfg.LoginToken = cred.LoginToken()
		return dnspod.NewDNSProviderConfig(cfg)
	case "alidns":
		cfg := alidns.NewDefaultConfig()
		cfg.APIKey = cred.Token
		cfg.SecretKey = cred.Secret
		return alidns.NewDNSProviderConfig(cfg)
	default:
		cfg := cloudflare.NewDefaultConfig()
		cfg.AuthToken = cred.Token
		return cloudflare.NewDNSProviderConfig(cfg)
	}
}

func dnsProviderName(provider string) string {
	switch provider {
	case "dnspod", "alidns":
		return provider
	default:
		return "cloudflare"
	}
}

type loggingDNSProvider struct {
	inner challenge.Provider
	job   *Job
}

func wrapDNSProvider(inner challenge.Provider, job *Job) challenge.Provider {
	if job == nil {
		return inner
	}
	return &loggingDNSProvider{inner: inner, job: job}
}

func (p *loggingDNSProvider) Present(domain, token, keyAuth string) error {
	record := strings.TrimSuffix(domain, ".")
	p.job.Info(fmt.Sprintf("添加 DNS TXT 记录: _acme-challenge.%s", record))
	if err := p.inner.Present(domain, token, keyAuth); err != nil {
		p.job.Error(fmt.Sprintf("添加 DNS TXT 记录失败: %v", err))
		return err
	}
	p.job.Info("DNS TXT 记录已添加，等待 DNS 传播…")
	return nil
}

func (p *loggingDNSProvider) CleanUp(domain, token, keyAuth string) error {
	p.job.Info("清理 DNS TXT 验证记录…")
	return p.inner.CleanUp(domain, token, keyAuth)
}

func validateDNSCredentials(provider string, cred ddns.Credentials) error {
	if err := cred.Validate(dnsProviderName(provider)); err != nil {
		return err
	}
	if provider == "dnspod" && cred.TokenID == "" {
		return fmt.Errorf("请先在 DDNS 页面配置 DNSPod ID 和 Token")
	}
	if provider == "alidns" && cred.Secret == "" {
		return fmt.Errorf("请先在 DDNS 页面配置阿里云 AccessKey")
	}
	if provider != "dnspod" && provider != "alidns" && cred.Token == "" {
		return fmt.Errorf("请先在 DDNS 页面配置 Cloudflare API Token")
	}
	return nil
}
