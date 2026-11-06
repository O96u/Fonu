package ddns

import (
	"context"
	"strings"
)

// UpsertTXTRecord creates or updates a TXT record for ACME DNS-01.
func (d *DNSPod) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return d.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	return d.UpdateRecord(ctx, cred, zone, host, "TXT", value)
}

// DeleteTXTRecord removes TXT records matching value at fqdn (ACME cleanup).
func (d *DNSPod) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return d.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
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
	want := strings.TrimSpace(value)
	for _, rec := range records {
		if !strings.EqualFold(rec.Type, "TXT") || strings.TrimSpace(rec.Value) != want {
			continue
		}
		if _, err := client.Records.Delete(domainID, rec.ID); err != nil {
			return err
		}
	}
	return nil
}
