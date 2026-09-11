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
	_, _, err := client.Domains.List()
	return err
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

func (d *DNSPod) domainID(client *dnspod.Client, rootDomain string) (string, error) {
	domains, _, err := client.Domains.List()
	if err != nil {
		return "", err
	}
	for _, domain := range domains {
		if domain.Name == rootDomain {
			return domain.ID.String(), nil
		}
	}
	return "", fmt.Errorf("未找到域名 %s 对应的 DNSPod 域名", rootDomain)
}

func (d *DNSPod) apiClient(cred Credentials) *dnspod.Client {
	params := dnspod.CommonParams{LoginToken: cred.LoginToken(), Format: "json"}
	client := dnspod.NewClient(params)
	client.HTTPClient = d.client
	return client
}
