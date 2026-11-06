package acme

import (
	"context"
	"fmt"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"

	"github.com/fonu/fonu/internal/ddns"
)

type tencentcloudDNS struct {
	provider *ddns.TencentCloud
	cred     ddns.Credentials
}

func newTencentCloudDNSProvider(cred ddns.Credentials) (challenge.Provider, error) {
	if err := cred.Validate("tencentcloud"); err != nil {
		return nil, err
	}
	if cred.Secret == "" {
		return nil, fmt.Errorf("请先在 DDNS 页面配置腾讯云 SecretKey")
	}
	return &tencentcloudDNS{
		provider: ddns.NewTencentCloud(),
		cred:     cred,
	}, nil
}

func (p *tencentcloudDNS) Present(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.UpsertTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}

func (p *tencentcloudDNS) CleanUp(domain, _ string, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	return p.provider.DeleteTXTRecord(context.Background(), p.cred, fqdn, info.Value)
}
