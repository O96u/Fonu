package acme

import (
	"context"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/fonu/fonu/internal/ddns"
)

type volcengineDNS struct {
	provider *ddns.Volcengine
	cred     ddns.Credentials
}

func newVolcengineDNSProvider(cred ddns.Credentials) (challenge.Provider, error) {
	if err := cred.Validate("volcengine"); err != nil {
		return nil, err
	}
	return &volcengineDNS{
		provider: ddns.NewVolcengine(),
		cred:     cred,
	}, nil
}

func (p *volcengineDNS) Present(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.UpsertTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}

func (p *volcengineDNS) CleanUp(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.DeleteTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}
