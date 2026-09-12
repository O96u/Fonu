package acme

import (
	"fmt"

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
