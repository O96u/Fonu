package ddns

import (
	"context"
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
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
	return t.ensureTXTRecord(ctx, cred, zone, host, value)
}

func (t *TencentCloud) ensureTXTRecord(ctx context.Context, cred Credentials, zone, host, value string) error {
	client, err := t.client(cred)
	if err != nil {
		return err
	}
	zoneInfo, err := t.domainInfo(client, zone)
	if err != nil {
		return err
	}
	return ensureTencentTXT(ctx,client,zoneInfo,tencentSubdomain(host),value)
}

func ensureTencentTXT(ctx context.Context,client tencentRecordAPI,zoneInfo *dnspod.DomainListItem,subDomain,value string)error {
	if strings.TrimSpace(value)=="" {return fmt.Errorf("empty ACME TXT value")}
	records, err := listTencentRecords(ctx,client,zoneInfo,subDomain,"TXT")
	if err != nil {
		return err
	}
	want := strings.TrimSpace(value)
	var existing []string
	for _, rec := range records {
		if rec.Value != nil {
			existing = append(existing, dnspodTXTValue(*rec.Value))
		}
	}
	if txtValuePresent(existing, want) {
		return nil
	}
	req := dnspod.NewCreateRecordRequest()
	req.Domain = zoneInfo.Name
	req.DomainId = zoneInfo.DomainId
	req.SubDomain = common.StringPtr(subDomain)
	req.RecordType = common.StringPtr("TXT")
	req.RecordLine = common.StringPtr("默认")
	req.Value = common.StringPtr(want)
	// TTL is optional; let DNSPod choose the valid value for this plan.
	_, err = client.CreateRecordWithContext(ctx,req)
	return wrapTencentErr(err)
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
	return deleteTencentTXT(ctx,client,zoneInfo,tencentSubdomain(host),value)
}

func deleteTencentTXT(ctx context.Context,client tencentRecordAPI,zoneInfo *dnspod.DomainListItem,subDomain,value string)error {
	if strings.TrimSpace(value)=="" {return fmt.Errorf("empty ACME TXT value")}
	records, err := listTencentRecords(ctx,client,zoneInfo,subDomain,"TXT")
	if err != nil {
		return err
	}
	want := strings.TrimSpace(value)
	for _, rec := range records {
		if rec.Value == nil || dnspodTXTValue(*rec.Value) != want {
			continue
		}
		if rec.RecordId == nil {
			continue
		}
		req := dnspod.NewDeleteRecordRequest()
		req.Domain = zoneInfo.Name
		req.DomainId = zoneInfo.DomainId
		req.RecordId = rec.RecordId
		if _, err := client.DeleteRecordWithContext(ctx,req); err != nil {
			return wrapTencentErr(err)
		}
	}
	return nil
}
