package ddns

import (
	"context"

	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)

// UpsertTXTRecord adds one ACME DNS-01 token without replacing other TXT values
// on the same name. Apex and wildcard challenges share that name.
func (h *HuaweiCloud) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zoneName, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return h.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	client, err := h.client(cred)
	if err != nil {
		return err
	}
	zone, err := h.findZone(client, zoneName)
	if err != nil {
		return err
	}
	return h.ensureTXTRecord(client, huaweiDeref(zone.Id), fqdnForRecord(zoneName, host), value)
}

// DeleteTXTRecord removes one ACME token and keeps any other TXT values.
func (h *HuaweiCloud) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	zoneName, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return h.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	client, err := h.client(cred)
	if err != nil {
		return err
	}
	zone, err := h.findZone(client, zoneName)
	if err != nil {
		return err
	}
	zoneID := huaweiDeref(zone.Id)
	name := fqdnForRecord(zoneName, host)
	rec, err := h.findRecordSet(client, zoneID, name, "TXT")
	if err != nil || rec == nil || rec.Id == nil {
		return err
	}
	want := huaweiPlainValue(value)
	var kept []string
	if rec.Records != nil {
		for _, item := range *rec.Records {
			if huaweiPlainValue(item) == want {
				continue
			}
			kept = append(kept, huaweiQuoteTXT(item))
		}
	}
	fqdnName := huaweiFQDN(name)
	if len(kept) == 0 {
		_, err = client.DeleteRecordSet(&model.DeleteRecordSetRequest{
			ZoneId:      zoneID,
			RecordsetId: *rec.Id,
		})
		return wrapHuaweiErr(err)
	}
	ttl := int32(120)
	if rec.Ttl != nil && *rec.Ttl > 0 {
		ttl = *rec.Ttl
	}
	_, err = client.UpdateRecordSet(&model.UpdateRecordSetRequest{
		ZoneId:      zoneID,
		RecordsetId: *rec.Id,
		Body: &model.UpdateRecordSetReq{
			Name:    &fqdnName,
			Type:    huaweiStr("TXT"),
			Ttl:     &ttl,
			Records: &kept,
		},
	})
	return wrapHuaweiErr(err)
}

func (h *HuaweiCloud) ensureTXTRecord(client *dns.DnsClient, zoneID, fqdn, value string) error {
	rec, err := h.findRecordSet(client, zoneID, fqdn, "TXT")
	if err != nil {
		return err
	}
	want := huaweiPlainValue(value)
	var existing []string
	if rec != nil && rec.Records != nil {
		for _, item := range *rec.Records {
			existing = append(existing, huaweiPlainValue(item))
		}
	}
	if txtValuePresent(existing, want) {
		return nil
	}
	name := huaweiFQDN(fqdn)
	ttl := int32(120)
	if rec != nil && rec.Id != nil {
		records := []string{huaweiQuoteTXT(want)}
		for _, item := range existing {
			records = append(records, huaweiQuoteTXT(item))
		}
		if rec.Ttl != nil && *rec.Ttl > 0 {
			ttl = *rec.Ttl
		}
		_, err = client.UpdateRecordSet(&model.UpdateRecordSetRequest{
			ZoneId:      zoneID,
			RecordsetId: *rec.Id,
			Body: &model.UpdateRecordSetReq{
				Name:    &name,
				Type:    huaweiStr("TXT"),
				Ttl:     &ttl,
				Records: &records,
			},
		})
		return wrapHuaweiErr(err)
	}
	_, err = client.CreateRecordSet(&model.CreateRecordSetRequest{
		ZoneId: zoneID,
		Body: &model.CreateRecordSetRequestBody{
			Name:    name,
			Type:    "TXT",
			Ttl:     &ttl,
			Records: []string{huaweiQuoteTXT(want)},
		},
	})
	return wrapHuaweiErr(err)
}
