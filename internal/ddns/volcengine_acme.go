package ddns

import (
	"context"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/dns"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

// UpsertTXTRecord creates or updates a TXT record for ACME DNS-01.
func (v *Volcengine) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return v.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	client, err := v.client(cred)
	if err != nil {
		return err
	}
	zid, err := v.zoneID(ctx, client, zone)
	if err != nil {
		return err
	}
	hostName := volcengineHost(host)
	records, err := v.listRecords(ctx, client, zid, hostName, "TXT")
	if err != nil {
		return err
	}
	var existing []string
	for _, rec := range records {
		if rec.Value != nil {
			existing = append(existing, *rec.Value)
		}
	}
	if txtValuePresent(existing, value) {
		return nil
	}
	_, err = client.CreateRecordWithContext(ctx, &dns.CreateRecordInput{
		ZID:   volcengine.Int64(zid),
		Host:  volcengine.String(hostName),
		Type:  volcengine.String("TXT"),
		Value: volcengine.String(strings.TrimSpace(value)),
		Line:  volcengine.String("default"),
		TTL:   volcengine.Int32(120),
	})
	return wrapVolcengineErr(err)
}

// DeleteTXTRecord removes TXT records matching value at fqdn (ACME cleanup).
func (v *Volcengine) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return v.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	return v.deleteTXTRecords(ctx, cred, zone, host, value)
}

func (v *Volcengine) deleteTXTRecords(ctx context.Context, cred Credentials, rootDomain, recordName, value string) error {
	client, err := v.client(cred)
	if err != nil {
		return err
	}
	zid, err := v.zoneID(ctx, client, rootDomain)
	if err != nil {
		return err
	}
	host := volcengineHost(recordName)
	records, err := v.listRecords(ctx, client, zid, host, "TXT")
	if err != nil {
		return err
	}
	for _, rec := range records {
		if rec.Value == nil || strings.TrimSpace(*rec.Value) != strings.TrimSpace(value) {
			continue
		}
		if rec.RecordID == nil {
			continue
		}
		if _, err := client.DeleteRecordWithContext(ctx, &dns.DeleteRecordInput{
			RecordID: rec.RecordID,
		}); err != nil {
			return wrapVolcengineErr(err)
		}
	}
	return nil
}

func (v *Volcengine) listRecords(ctx context.Context, client *dns.DNS, zid int64, host, recordType string) ([]*dns.RecordForListRecordsOutput, error) {
	var out []*dns.RecordForListRecordsOutput
	var page int32 = 1
	const pageSize int32 = 100
	for {
		resp, err := client.ListRecordsWithContext(ctx, &dns.ListRecordsInput{
			ZID:        volcengine.Int64(zid),
			Host:       volcengine.String(host),
			Type:       volcengine.String(recordType),
			PageNumber: volcengine.Int32(page),
			PageSize:   volcengine.Int32(pageSize),
		})
		if err != nil {
			return nil, wrapVolcengineErr(err)
		}
		out = append(out, resp.Records...)
		total := int32(0)
		if resp.TotalCount != nil {
			total = *resp.TotalCount
		}
		if len(resp.Records) == 0 || page*pageSize >= total {
			break
		}
		page++
	}
	return out, nil
}
