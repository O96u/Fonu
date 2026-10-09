package ddns

import (
	"context"
)

// UpsertTXTRecord 创建或更新 TXT 记录，用于 ACME DNS-01 验证。
func (d *DNSHE) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, _, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return d.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	subID, err := d.subdomainID(ctx, cred, zone)
	if err != nil {
		return err
	}
	return d.ensureTXTRecord(ctx, cred, subID, zone, fqdn, value)
}

// DeleteTXTRecord 删除 fqdn 上内容匹配的 TXT 记录（ACME 清理）。
func (d *DNSHE) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, _, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return d.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	subID, err := d.subdomainID(ctx, cred, zone)
	if err != nil {
		return err
	}
	return d.deleteRecords(ctx, cred, subID, fqdn, "TXT", value)
}
