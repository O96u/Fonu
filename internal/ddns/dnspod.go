package ddns

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nrdcg/dnspod-go"
)

type DNSPod struct {
	client *http.Client
}

func NewDNSPod() *DNSPod {
	return &DNSPod{client: &http.Client{Timeout: 15 * time.Second}}
}

func (d *DNSPod) Name() string { return "dnspod" }

func (d *DNSPod) Verify(ctx context.Context, cred Credentials) error {
	client := d.apiClient(cred)
	domains, _, err := client.Domains.List()
	if err != nil {
		return dnsPodDomainListError(err)
	}
	if len(domains) == 0 {
		return fmt.Errorf("DNSPod 域名列表为空，请确认凭证所属账号已托管域名")
	}
	return nil
}

func (d *DNSPod) HasZone(ctx context.Context, cred Credentials, zone string) (bool, error) {
	// Only an explicit successful list without this zone means the candidate is absent.
	domains, _, err := d.apiClient(cred).Domains.List()
	if err != nil {
		return false, dnsPodDomainListError(err)
	}
	if len(domains) == 0 {
		return false, fmt.Errorf("DNSPod 域名列表为空，请确认凭证所属账号已托管域名")
	}
	for _, domain := range domains {
		if strings.EqualFold(domain.Name, zone) {
			return true, nil
		}
	}
	return false, nil
}

func (d *DNSPod) GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error) {
	client := d.apiClient(cred)
	domainID, err := d.domainID(client, rootDomain)
	if err != nil {
		return "", err
	}
	subDomain := subDomainForRecord(recordName)
	records, _, err := client.Records.List(domainID, subDomain)
	if err != nil {
		return "", err
	}
	for _, rec := range records {
		if strings.EqualFold(rec.Type, recordType) {
			return rec.Value, nil
		}
	}
	return "", nil
}

func (d *DNSPod) UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, ip string) error {
	client := d.apiClient(cred)
	domainID, err := d.domainID(client, rootDomain)
	if err != nil {
		return err
	}
	subDomain := subDomainForRecord(recordName)
	records, _, err := client.Records.List(domainID, subDomain)
	if err != nil {
		return err
	}
	for _, rec := range records {
		if strings.EqualFold(rec.Type, recordType) {
			_, _, err := client.Records.Update(domainID, rec.ID, dnspod.Record{
				Type:  recordType,
				Name:  subDomain,
				Value: ip,
				Line:  rec.Line,
				TTL:   rec.TTL,
			})
			return err
		}
	}
	_, _, err = client.Records.Create(domainID, dnspod.Record{
		Type:  recordType,
		Name:  subDomain,
		Value: ip,
		Line:  "默认",
		TTL:   strconv.Itoa(120),
	})
	return err
}

// removeTXTRecords deletes TXT records at host. If matchValue is non-empty, only that value is removed.
func (d *DNSPod) removeTXTRecords(ctx context.Context, cred Credentials, zone, host, matchValue string) error {
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
	want := strings.TrimSpace(matchValue)
	for _, rec := range records {
		if !strings.EqualFold(rec.Type, "TXT") {
			continue
		}
		if want != "" && dnspodTXTValue(rec.Value) != want {
			continue
		}
		if _, err := client.Records.Delete(domainID, rec.ID); err != nil {
			return err
		}
	}
	return nil
}

func dnspodTXTValue(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func (d *DNSPod) domainID(client *dnspod.Client, rootDomain string) (string, error) {
	domains, _, err := client.Domains.List()
	if err != nil {
		return "", dnsPodDomainListError(err)
	}
	if len(domains) == 0 {
		return "", fmt.Errorf("DNSPod 域名列表为空，请确认凭证所属账号已托管域名")
	}
	for _, domain := range domains {
		if strings.EqualFold(domain.Name, rootDomain) {
			return domain.ID.String(), nil
		}
	}
	return "", fmt.Errorf("未找到域名 %s 对应的 DNSPod 域名", rootDomain)
}

// Domain.List errors concern the account as a whole, not a candidate zone.
// Keep permission/transport failures fatal; an empty account cannot be fixed by trying a parent zone.
func dnsPodDomainListError(err error) error {
	if strings.Contains(err.Error(), "记录列表为空") || strings.Contains(err.Error(), "域名列表为空") {
		return fmt.Errorf("DNSPod 域名列表为空，请确认使用 DNSPod API Token、账号权限及域名托管情况：%w", err)
	}
	return fmt.Errorf("DNSPod 获取账号域名列表失败：%w", err)
}

func (d *DNSPod) apiClient(cred Credentials) *dnspod.Client {
	params := dnspod.CommonParams{LoginToken: cred.LoginToken(), Format: "json"}
	client := dnspod.NewClient(params)
	client.HTTPClient = d.client
	return client
}
