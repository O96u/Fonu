package ddns

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/dns"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
	"github.com/volcengine/volcengine-go-sdk/volcengine/volcengineerr"
)

const volcengineDNSRegion = "cn-beijing"

type Volcengine struct{}

func NewVolcengine() *Volcengine { return &Volcengine{} }

func (v *Volcengine) Name() string { return "volcengine" }

func (v *Volcengine) Verify(ctx context.Context, cred Credentials) error {
	client, err := v.client(cred)
	if err != nil {
		return err
	}
	_, err = client.ListZonesWithContext(ctx, &dns.ListZonesInput{
		PageNumber: volcengine.Int32(1),
		PageSize:   volcengine.Int32(1),
	})
	return wrapVolcengineErr(err)
}

func (v *Volcengine) HasZone(ctx context.Context, cred Credentials, zone string) (bool, error) {
	client, err := v.client(cred)
	if err != nil {
		return false, err
	}
	_, err = v.zoneID(ctx, client, zone)
	return zoneLookupOK(err)
}

func (v *Volcengine) GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error) {
	client, err := v.client(cred)
	if err != nil {
		return "", err
	}
	zid, err := v.zoneID(ctx, client, rootDomain)
	if err != nil {
		return "", err
	}
	rec, err := v.findRecord(ctx, client, zid, volcengineHost(recordName), recordType)
	if err != nil {
		return "", err
	}
	if rec != nil && rec.Value != nil {
		return *rec.Value, nil
	}
	return "", nil
}

func (v *Volcengine) UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, ip string) error {
	client, err := v.client(cred)
	if err != nil {
		return err
	}
	zid, err := v.zoneID(ctx, client, rootDomain)
	if err != nil {
		return err
	}
	host := volcengineHost(recordName)
	existing, err := v.findRecord(ctx, client, zid, host, recordType)
	if err != nil {
		return err
	}
	if existing != nil && existing.RecordID != nil {
		line := volcengineLine(existing)
		_, err = client.UpdateRecordWithContext(ctx, &dns.UpdateRecordInput{
			RecordID: existing.RecordID,
			Host:     volcengine.String(host),
			Line:     volcengine.String(line),
			Type:     volcengine.String(recordType),
			Value:    volcengine.String(ip),
			TTL:      volcengine.Int32(600),
		})
		return wrapVolcengineErr(err)
	}

	_, err = client.CreateRecordWithContext(ctx, &dns.CreateRecordInput{
		ZID:   volcengine.Int64(zid),
		Host:  volcengine.String(host),
		Type:  volcengine.String(recordType),
		Value: volcengine.String(ip),
		Line:  volcengine.String("default"),
		TTL:   volcengine.Int32(600),
	})
	return wrapVolcengineErr(err)
}

func (v *Volcengine) client(cred Credentials) (*dns.DNS, error) {
	if cred.Token == "" || cred.Secret == "" {
		return nil, fmt.Errorf("火山引擎 DNS 凭证未配置")
	}
	cfg := volcengine.NewConfig().
		WithRegion(volcengineDNSRegion).
		WithCredentials(credentials.NewStaticCredentials(cred.Token, cred.Secret, ""))
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, err
	}
	return dns.New(sess), nil
}

func (v *Volcengine) zoneID(ctx context.Context, client *dns.DNS, rootDomain string) (int64, error) {
	rootDomain = strings.ToLower(strings.TrimSpace(rootDomain))
	var page int32 = 1
	const pageSize int32 = 100
	for {
		out, err := client.ListZonesWithContext(ctx, &dns.ListZonesInput{
			Key:        volcengine.String(rootDomain),
			PageNumber: volcengine.Int32(page),
			PageSize:   volcengine.Int32(pageSize),
		})
		if err != nil {
			return 0, wrapVolcengineErr(err)
		}
		for _, zone := range out.Zones {
			if zone.ZoneName != nil && strings.EqualFold(*zone.ZoneName, rootDomain) && zone.ZID != nil {
				return int64(*zone.ZID), nil
			}
		}
		total := int32(0)
		if out.Total != nil {
			total = *out.Total
		}
		if len(out.Zones) == 0 || page*pageSize >= total {
			break
		}
		page++
	}
	return 0, fmt.Errorf("未找到域名 %s 对应的火山引擎 DNS 域名", rootDomain)
}

func (v *Volcengine) findRecord(ctx context.Context, client *dns.DNS, zid int64, host, recordType string) (*dns.RecordForListRecordsOutput, error) {
	var page int32 = 1
	const pageSize int32 = 100
	for {
		out, err := client.ListRecordsWithContext(ctx, &dns.ListRecordsInput{
			ZID:        volcengine.Int64(zid),
			Host:       volcengine.String(host),
			Type:       volcengine.String(recordType),
			PageNumber: volcengine.Int32(page),
			PageSize:   volcengine.Int32(pageSize),
		})
		if err != nil {
			return nil, wrapVolcengineErr(err)
		}
		for _, rec := range out.Records {
			if rec.Host != nil && *rec.Host == host && rec.Type != nil && strings.EqualFold(*rec.Type, recordType) {
				return rec, nil
			}
		}
		total := int32(0)
		if out.TotalCount != nil {
			total = *out.TotalCount
		}
		if len(out.Records) == 0 || page*pageSize >= total {
			break
		}
		page++
	}
	return nil, nil
}

func volcengineHost(recordName string) string {
	return subDomainForRecord(recordName)
}

func volcengineLine(rec *dns.RecordForListRecordsOutput) string {
	if rec != nil && rec.Line != nil && strings.TrimSpace(*rec.Line) != "" {
		return *rec.Line
	}
	return "default"
}

func wrapVolcengineErr(err error) error {
	if err == nil {
		return nil
	}
	var sdkErr volcengineerr.RequestFailure
	if errors.As(err, &sdkErr) {
		return fmt.Errorf("火山引擎 DNS API 错误：%s", sdkErr.Message())
	}
	var apiErr volcengineerr.Error
	if errors.As(err, &apiErr) {
		return fmt.Errorf("火山引擎 DNS API 错误：%s", apiErr.Message())
	}
	return err
}
