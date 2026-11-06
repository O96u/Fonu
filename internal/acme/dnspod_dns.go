package acme

import (
	"context"
	"fmt"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/fonu/fonu/internal/ddns"
)

type dnspodDNS struct {
	provider *ddns.DNSPod
	cred     ddns.Credentials
}

func newDNSPodDNSProvider(cred ddns.Credentials) (challenge.Provider, error) {
	if err := cred.Validate("dnspod"); err != nil {
		return nil, err
	}
	if cred.TokenID == "" {
		return nil, fmt.Errorf("请先在 DDNS 页面配置 DNSPod ID 和 Token")
	}
	return &dnspodDNS{
		provider: ddns.NewDNSPod(),
		cred:     cred,
	}, nil
}

func (p *dnspodDNS) Present(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.UpsertTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}

func (p *dnspodDNS) CleanUp(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.DeleteTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}
