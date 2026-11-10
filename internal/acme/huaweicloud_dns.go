package acme

import (
	"context"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/fonu/fonu/internal/ddns"
)

type huaweiCloudDNS struct {
	provider *ddns.HuaweiCloud
	cred     ddns.Credentials
}

func newHuaweiCloudDNSProvider(cred ddns.Credentials) (challenge.Provider, error) {
	if err := cred.Validate("huaweicloud"); err != nil {
		return nil, err
	}
	return &huaweiCloudDNS{
		provider: ddns.NewHuaweiCloud(),
		cred:     cred,
	}, nil
}

func (p *huaweiCloudDNS) Present(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.UpsertTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}

func (p *huaweiCloudDNS) CleanUp(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.DeleteTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}
