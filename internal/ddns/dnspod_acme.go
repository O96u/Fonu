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
	// DNSPod 同主机名下可有多条 TXT（多线路或历史残留）；只改第一条会导致权威 NS 仍返回旧 token。
	if err := d.removeTXTRecords(ctx, cred, zone, host, ""); err != nil {
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
	return d.removeTXTRecords(ctx, cred, zone, host, strings.TrimSpace(value))
}
