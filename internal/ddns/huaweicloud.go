package ddns

import (
	"context"
	"fmt"
	"strings"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
	region "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/region"
)

// Public DNS zones are global. cn-north-4 is the default endpoint for the public API.
const huaweiDNSRegion = "cn-north-4"

type HuaweiCloud struct{}

func NewHuaweiCloud() *HuaweiCloud { return &HuaweiCloud{} }

func (h *HuaweiCloud) Name() string { return "huaweicloud" }

func (h *HuaweiCloud) Verify(ctx context.Context, cred Credentials) error {
	client, err := h.client(cred)
	if err != nil {
		return err
	}
	req := &model.ListPublicZonesRequest{
		Type:  huaweiStr("public"),
		Limit: huaweiInt32(1),
	}
	_, err = client.ListPublicZones(req)
	return wrapHuaweiErr(err)
}

func (h *HuaweiCloud) HasZone(ctx context.Context, cred Credentials, zone string) (bool, error) {
	client, err := h.client(cred)
	if err != nil {
		return false, err
	}
	_, err = h.findZone(client, zone)
	return zoneLookupOK(err)
}

func (h *HuaweiCloud) GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error) {
	client, err := h.client(cred)
	if err != nil {
		return "", err
	}
	zone, err := h.findZone(client, rootDomain)
	if err != nil {
		return "", err
	}
	rec, err := h.findRecordSet(client, huaweiDeref(zone.Id), fqdnForRecord(rootDomain, recordName), recordType)
	if err != nil || rec == nil || rec.Records == nil || len(*rec.Records) == 0 {
		return "", err
	}
	return huaweiPlainValue((*rec.Records)[0]), nil
}

func (h *HuaweiCloud) UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, ip string) error {
	client, err := h.client(cred)
	if err != nil {
		return err
	}
	zone, err := h.findZone(client, rootDomain)
	if err != nil {
		return err
	}
	name := huaweiFQDN(fqdnForRecord(rootDomain, recordName))
	zoneID := huaweiDeref(zone.Id)
	existing, err := h.findRecordSet(client, zoneID, strings.TrimSuffix(name, "."), recordType)
	if err != nil {
		return err
	}
	value := huaweiStoredValue(recordType, ip)
	ttl := int32(120)
	if existing != nil && existing.Id != nil {
		records := []string{value}
		_, err = client.UpdateRecordSet(&model.UpdateRecordSetRequest{
			ZoneId:      zoneID,
			RecordsetId: *existing.Id,
			Body: &model.UpdateRecordSetReq{
				Name:    &name,
				Type:    huaweiStr(recordType),
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
			Type:    recordType,
			Ttl:     &ttl,
			Records: []string{value},
		},
	})
	return wrapHuaweiErr(err)
}

func (h *HuaweiCloud) client(cred Credentials) (*dns.DnsClient, error) {
	if cred.Token == "" || cred.Secret == "" {
		return nil, fmt.Errorf("华为云 DNS 凭证未配置")
	}
	auth, err := basic.NewCredentialsBuilder().
		WithAk(cred.Token).
		WithSk(cred.Secret).
		SafeBuild()
	if err != nil {
		return nil, err
	}
	reg, err := region.SafeValueOf(huaweiDNSRegion)
	if err != nil {
		return nil, err
	}
	hc, err := dns.DnsClientBuilder().
		WithRegion(reg).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return nil, err
	}
	return dns.NewDnsClient(hc), nil
}

func (h *HuaweiCloud) findZone(client *dns.DnsClient, rootDomain string) (*model.PublicZoneResp, error) {
	rootDomain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(rootDomain), "."))
	req := &model.ListPublicZonesRequest{
		Type:       huaweiStr("public"),
		Name:       huaweiStr(rootDomain),
		SearchMode: huaweiStr("equal"),
		Limit:      huaweiInt32(20),
	}
	resp, err := client.ListPublicZones(req)
	if err != nil {
		return nil, wrapHuaweiErr(err)
	}
	if resp.Zones != nil {
		for i := range *resp.Zones {
			zone := &(*resp.Zones)[i]
			if zone.Name != nil && strings.EqualFold(strings.TrimSuffix(*zone.Name, "."), rootDomain) && zone.Id != nil {
				return zone, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到域名 %s 对应的华为云 DNS 域名", rootDomain)
}

func (h *HuaweiCloud) findRecordSet(client *dns.DnsClient, zoneID, fqdn, recordType string) (*model.ListRecordSets, error) {
	name := huaweiFQDN(fqdn)
	req := &model.ListRecordSetsByZoneRequest{
		ZoneId:     zoneID,
		Name:       huaweiStr(strings.TrimSuffix(name, ".")),
		Type:       huaweiStr(recordType),
		SearchMode: huaweiStr("equal"),
		Limit:      huaweiInt32(50),
	}
	resp, err := client.ListRecordSetsByZone(req)
	if err != nil {
		return nil, wrapHuaweiErr(err)
	}
	if resp.Recordsets == nil {
		return nil, nil
	}
	for i := range *resp.Recordsets {
		rec := &(*resp.Recordsets)[i]
		if rec.Name == nil || rec.Type == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSuffix(*rec.Name, "."), strings.TrimSuffix(name, ".")) && strings.EqualFold(*rec.Type, recordType) {
			return rec, nil
		}
	}
	return nil, nil
}

func huaweiFQDN(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return ""
	}
	return name + "."
}

func huaweiStoredValue(recordType, value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(recordType, "TXT") {
		return huaweiQuoteTXT(value)
	}
	return value
}

func huaweiQuoteTXT(value string) string {
	value = huaweiPlainValue(value)
	return `"` + value + `"`
}

func huaweiPlainValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}
	return value
}

func huaweiStr(v string) *string { return &v }

func huaweiInt32(v int32) *int32 { return &v }

func huaweiDeref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func wrapHuaweiErr(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return fmt.Errorf("华为云 DNS 请求失败")
	}
	return fmt.Errorf("华为云 DNS: %s", msg)
}
