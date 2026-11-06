package ddns

import (
	"context"
	"strings"

	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

// UpsertTXTRecord creates or updates a TXT record for ACME DNS-01.
func (t *TencentCloud) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return t.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	return t.UpdateRecord(ctx, cred, zone, host, "TXT", value)
}

// DeleteTXTRecord removes TXT records matching value at fqdn (ACME cleanup).
func (t *TencentCloud) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return t.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	client, err := t.client(cred)
	if err != nil {
		return err
	}
	zoneInfo, err := t.domainInfo(client, zone)
	if err != nil {
		return err
	}
	subDomain := tencentSubdomain(host)
	records, err := t.listRecords(client, zoneInfo, subDomain, "TXT")
	if err != nil {
		return err
	}
	want := strings.TrimSpace(value)
	for _, rec := range records {
		if rec.Value == nil || strings.TrimSpace(*rec.Value) != want {
			continue
		}
		if rec.RecordId == nil {
			continue
		}
		req := dnspod.NewDeleteRecordRequest()
		req.Domain = zoneInfo.Name
		req.DomainId = zoneInfo.DomainId
		req.RecordId = rec.RecordId
		if _, err := client.DeleteRecord(req); err != nil {
			return wrapTencentErr(err)
		}
	}
	return nil
}
