package acme

import (
	"context"
	"fmt"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/fonu/fonu/internal/ddns"
)

type alidnsDNS struct {
	provider *ddns.AliDNS
	cred     ddns.Credentials
}

func newAliDNSDNSProvider(cred ddns.Credentials) (challenge.Provider, error) {
	if err := cred.Validate("alidns"); err != nil {
		return nil, err
	}
	if cred.Secret == "" {
		return nil, fmt.Errorf("请先在 DDNS 页面配置阿里云 AccessKey")
	}
	return &alidnsDNS{
		provider: ddns.NewAliDNS(),
		cred:     cred,
	}, nil
}

func (p *alidnsDNS) Present(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.UpsertTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}

func (p *alidnsDNS) CleanUp(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.DeleteTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}
