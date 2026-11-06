package ddns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// UpsertTXTRecord creates or updates a TXT record for ACME DNS-01.
func (c *Cloudflare) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return c.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	return c.UpdateRecord(ctx, cred, zone, host, "TXT", value)
}

// DeleteTXTRecord removes TXT records matching value at fqdn (ACME cleanup).
func (c *Cloudflare) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return c.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	zoneID, err := c.zoneID(ctx, cred.Token, zone)
	if err != nil {
		return err
	}
	recordFQDN := fqdnForRecord(zone, host)
	body, err := c.request(ctx, cred.Token, http.MethodGet, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=TXT&name=%s", zoneID, recordFQDN), nil)
	if err != nil {
		return err
	}
	var resp cfRecordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	want := strings.TrimSpace(value)
	for _, rec := range resp.Result {
		if strings.TrimSpace(rec.Content) != want {
			continue
		}
		if rec.ID == "" {
			continue
		}
		if _, err := c.request(ctx, cred.Token, http.MethodDelete, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, rec.ID), nil); err != nil {
			return err
		}
	}
	return nil
}
