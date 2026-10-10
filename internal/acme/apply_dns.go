package acme

import (
	"strings"

	"github.com/fonu/fonu/internal/ddns"
)

// InlineDNS carries DNS-01 credentials without requiring a DDNS sync task.
type InlineDNS struct {
	Provider   string `json:"provider"`
	APIToken   string `json:"api_token"`
	APITokenID string `json:"api_token_id"`
	APISecret  string `json:"api_secret"`
}

func (d InlineDNS) HasValues() bool {
	return ddns.CredentialsFromSave(ddns.SaveInput{
		Provider:   d.Provider,
		APIToken:   d.APIToken,
		APITokenID: d.APITokenID,
		APISecret:  d.APISecret,
	}).HasValues()
}

func (d InlineDNS) credentials() ddns.Credentials {
	provider := strings.TrimSpace(d.Provider)
	if provider == "" {
		provider = "cloudflare"
	}
	cred := ddns.CredentialsFromSave(ddns.SaveInput{
		Provider:   provider,
		APIToken:   d.APIToken,
		APITokenID: d.APITokenID,
		APISecret:  d.APISecret,
	})
	cred.Provider = provider
	return cred
}

func InferDNSZonesFromDomains(domains []string) []string {
	seen := map[string]struct{}{}
	var zones []string
	for _, domain := range domains {
		base := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(domain, "*.")))
		parts := strings.Split(base, ".")
		if len(parts) < 2 {
			continue
		}
		zone := strings.Join(parts[len(parts)-2:], ".")
		if _, ok := seen[zone]; ok {
			continue
		}
		seen[zone] = struct{}{}
		zones = append(zones, zone)
	}
	return zones
}
