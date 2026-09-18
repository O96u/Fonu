package ddns

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	errorsdk "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

type TencentCloud struct{}

func NewTencentCloud() *TencentCloud { return &TencentCloud{} }

func (t *TencentCloud) Name() string { return "tencentcloud" }

func (t *TencentCloud) Verify(ctx context.Context, cred Credentials) error {
	client, err := t.client(cred)
	if err != nil {
		return err
	}
	req := dnspod.NewDescribeDomainListRequest()
	req.Limit = common.Int64Ptr(1)
	_, err = client.DescribeDomainList(req)
	return wrapTencentErr(err)
}

func (t *TencentCloud) HasZone(ctx context.Context, cred Credentials, zone string) (bool, error) {
	client, err := t.client(cred)
	if err != nil {
		return false, err
	}
	_, err = t.domainInfo(client, zone)
	return zoneLookupOK(wrapTencentErr(err))
}

func (t *TencentCloud) GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error) {
	client, err := t.client(cred)
	if err != nil {
		return "", err
	}
	zone, err := t.domainInfo(client, rootDomain)
	if err != nil {
		return "", err
	}
	records, err := t.listRecords(client, zone, tencentSubdomain(recordName), recordType)
	if err != nil {
		return "", err
	}
	for _, rec := range records {
		if rec.Value != nil {
			return *rec.Value, nil
		}
	}
	return "", nil
}

func (t *TencentCloud) UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, ip string) error {
	client, err := t.client(cred)
	if err != nil {
		return err
	}
	zone, err := t.domainInfo(client, rootDomain)
	if err != nil {
		return err
	}
	subDomain := tencentSubdomain(recordName)
	records, err := t.listRecords(client, zone, subDomain, recordType)
	if err != nil {
		return err
	}
	if len(records) > 0 && records[0].RecordId != nil {
		req := dnspod.NewModifyRecordRequest()
		req.Domain = zone.Name
		req.DomainId = zone.DomainId
		req.RecordId = records[0].RecordId
		req.SubDomain = common.StringPtr(subDomain)
		req.RecordType = common.StringPtr(recordType)
		req.RecordLine = common.StringPtr(tencentRecordLine(records[0]))
		req.Value = common.StringPtr(ip)
		req.TTL = common.Uint64Ptr(600)
		_, err = client.ModifyRecord(req)
		return wrapTencentErr(err)
	}

	req := dnspod.NewCreateRecordRequest()
	req.Domain = zone.Name
	req.DomainId = zone.DomainId
	req.SubDomain = common.StringPtr(subDomain)
	req.RecordType = common.StringPtr(recordType)
	req.RecordLine = common.StringPtr("默认")
	req.Value = common.StringPtr(ip)
	req.TTL = common.Uint64Ptr(600)
	_, err = client.CreateRecord(req)
	return wrapTencentErr(err)
}

func (t *TencentCloud) client(cred Credentials) (*dnspod.Client, error) {
	if cred.Token == "" || cred.Secret == "" {
		return nil, fmt.Errorf("腾讯云 DNS 凭证未配置")
	}
	credential := common.NewCredential(cred.Token, cred.Secret)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "dnspod.tencentcloudapi.com"
	return dnspod.NewClient(credential, "", cpf)
}

func (t *TencentCloud) domainInfo(client *dnspod.Client, rootDomain string) (*dnspod.DomainListItem, error) {
	rootDomain = strings.ToLower(strings.TrimSpace(rootDomain))
	req := dnspod.NewDescribeDomainListRequest()
	var offset int64
	for {
		req.Offset = common.Int64Ptr(offset)
		resp, err := client.DescribeDomainList(req)
		if err != nil {
			return nil, wrapTencentErr(err)
		}
		for _, zone := range resp.Response.DomainList {
			if zone.Name != nil && strings.EqualFold(*zone.Name, rootDomain) {
				return zone, nil
			}
			if zone.Punycode != nil && strings.EqualFold(*zone.Punycode, rootDomain) {
				return zone, nil
			}
		}
		total := int64(0)
		if resp.Response.DomainCountInfo != nil && resp.Response.DomainCountInfo.AllTotal != nil {
			total = int64(*resp.Response.DomainCountInfo.AllTotal)
		}
		offset += int64(len(resp.Response.DomainList))
		if offset >= total || len(resp.Response.DomainList) == 0 {
			break
		}
	}
	return nil, fmt.Errorf("未找到域名 %s 对应的腾讯云 DNS 域名", rootDomain)
}

func (t *TencentCloud) listRecords(client *dnspod.Client, zone *dnspod.DomainListItem, subDomain, recordType string) ([]*dnspod.RecordListItem, error) {
	req := dnspod.NewDescribeRecordListRequest()
	req.Domain = zone.Name
	req.DomainId = zone.DomainId
	req.Subdomain = common.StringPtr(subDomain)
	req.RecordType = common.StringPtr(recordType)
	resp, err := client.DescribeRecordList(req)
	if err != nil {
		var sdkErr *errorsdk.TencentCloudSDKError
		if errors.As(err, &sdkErr) && sdkErr.Code == dnspod.RESOURCENOTFOUND_NODATAOFRECORD {
			return nil, nil
		}
		return nil, wrapTencentErr(err)
	}
	if resp.Response == nil {
		return nil, nil
	}
	return resp.Response.RecordList, nil
}

func tencentSubdomain(recordName string) string {
	sub := subDomainForRecord(recordName)
	if sub == "@" {
		return "@"
	}
	return sub
}

func tencentRecordLine(rec *dnspod.RecordListItem) string {
	if rec != nil && rec.Line != nil && *rec.Line != "" {
		return *rec.Line
	}
	return "默认"
}

func wrapTencentErr(err error) error {
	if err == nil {
		return nil
	}
	var sdkErr *errorsdk.TencentCloudSDKError
	if errors.As(err, &sdkErr) {
		return fmt.Errorf("腾讯云 DNS API 错误：%s", sdkErr.Message)
	}
	return err
}
