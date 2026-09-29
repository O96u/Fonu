package acme

import (
	"context"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/fonu/fonu/internal/ddns"
)

type dnsheDNS struct {
	provider *ddns.DNSHE
	cred     ddns.Credentials
}

func newDNSHEDNSProvider(cred ddns.Credentials) (challenge.Provider, error) {
	if err := cred.Validate("dnshe"); err != nil {
		return nil, err
	}
	return &dnsheDNS{
		provider: ddns.NewDNSHE(),
		cred:     cred,
	}, nil
}

func (p *dnsheDNS) Present(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.UpsertTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}

func (p *dnsheDNS) CleanUp(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.DeleteTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}
