package ddns

import (
	"context"
	"strings"

	"github.com/nrdcg/dnspod-go"
)

// UpsertTXTRecord creates or updates a TXT record for ACME DNS-01.
func (d *DNSPod) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return d.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	return d.ensureTXTRecord(ctx, cred, zone, host, value)
}

func (d *DNSPod) ensureTXTRecord(ctx context.Context, cred Credentials, zone, host, value string) error {
	client := d.apiClient(cred)
	domainID, err := d.domainID(client, zone)
	if err != nil {
		return err
	}
	subDomain := subDomainForRecord(host)
	records, _, err := client.Records.List(domainID, subDomain)
	if err != nil {
		return err
	}
	var existing []string
	for _, rec := range records {
		if strings.EqualFold(rec.Type, "TXT") {
			existing = append(existing, dnspodTXTValue(rec.Value))
		}
	}
	if txtValuePresent(existing, value) {
		return nil
	}
	_, _, err = client.Records.Create(domainID, dnspod.Record{
		Type:  "TXT",
		Name:  subDomain,
		Value: strings.TrimSpace(value),
		Line:  "默认",
		TTL:   "600",
	})
	return err
}

// DeleteTXTRecord removes TXT records matching value at fqdn (ACME cleanup).
func (d *DNSPod) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return d.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	return d.removeTXTRecords(ctx, cred, zone, host, strings.TrimSpace(value))
}
