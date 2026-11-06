package ddns

import (
	"context"
	"encoding/json"
	"strings"
)

// UpsertTXTRecord creates or updates a TXT record for ACME DNS-01.
func (a *AliDNS) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return a.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	subDomain := aliRR(host)
	body, err := a.request(ctx, cred, map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": zone,
		"RRKeyWord":  subDomain,
		"Type":       "TXT",
		"PageNumber": "1",
		"PageSize":   "50",
	})
	if err != nil {
		return err
	}
	var resp alidnsRecordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	var existing []string
	for _, rec := range resp.DomainRecords.Record {
		if strings.EqualFold(rec.Type, "TXT") && rec.RR == subDomain {
			existing = append(existing, rec.Value)
		}
	}
	if txtValuePresent(existing, value) {
		return nil
	}
	_, err = a.request(ctx, cred, map[string]string{
		"Action":     "AddDomainRecord",
		"DomainName": zone,
		"RR":         subDomain,
		"Type":       "TXT",
		"Value":      strings.TrimSpace(value),
		"TTL":        "120",
	})
	return err
}

// DeleteTXTRecord removes TXT records matching value at fqdn (ACME cleanup).
func (a *AliDNS) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return a.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	subDomain := aliRR(host)
	body, err := a.request(ctx, cred, map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": zone,
		"RRKeyWord":  subDomain,
		"Type":       "TXT",
		"PageNumber": "1",
		"PageSize":   "50",
	})
	if err != nil {
		return err
	}
	var resp alidnsRecordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	want := strings.TrimSpace(value)
	for _, rec := range resp.DomainRecords.Record {
		if !strings.EqualFold(rec.Type, "TXT") || rec.RR != subDomain || strings.TrimSpace(rec.Value) != want {
			continue
		}
		if rec.RecordID == "" {
			continue
		}
		if _, err := a.request(ctx, cred, map[string]string{
			"Action":   "DeleteDomainRecord",
			"RecordId": rec.RecordID,
		}); err != nil {
			return err
		}
	}
	return nil
}
