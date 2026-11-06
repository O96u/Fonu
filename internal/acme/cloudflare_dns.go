package acme

import (
	"context"
	"fmt"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/fonu/fonu/internal/ddns"
)

type cloudflareDNS struct {
	provider *ddns.Cloudflare
	cred     ddns.Credentials
}

func newCloudflareDNSProvider(cred ddns.Credentials) (challenge.Provider, error) {
	if err := cred.Validate("cloudflare"); err != nil {
		return nil, err
	}
	if cred.Token == "" {
		return nil, fmt.Errorf("请先在 DDNS 页面配置 Cloudflare API Token")
	}
	return &cloudflareDNS{
		provider: ddns.NewCloudflare(),
		cred:     cred,
	}, nil
}

func (p *cloudflareDNS) Present(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.UpsertTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}

func (p *cloudflareDNS) CleanUp(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.DeleteTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}
