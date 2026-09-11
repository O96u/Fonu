package ddns

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const alidnsEndpoint = "https://alidns.aliyuncs.com/"

type AliDNS struct {
	client *http.Client
}

func NewAliDNS() *AliDNS {
	return &AliDNS{client: &http.Client{Timeout: 15 * time.Second}}
}

func (a *AliDNS) Name() string { return "alidns" }

func (a *AliDNS) Verify(ctx context.Context, cred Credentials) error {
	_, err := a.request(ctx, cred, map[string]string{
		"Action":     "DescribeDomains",
		"PageNumber": "1",
		"PageSize":   "1",
	})
	return err
}

func (a *AliDNS) GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error) {
	subDomain := aliRR(recordName)
	body, err := a.request(ctx, cred, map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": rootDomain,
		"RRKeyWord":  subDomain,
		"Type":       recordType,
		"PageNumber": "1",
		"PageSize":   "50",
	})
	if err != nil {
		return "", err
	}
	var resp alidnsRecordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	for _, rec := range resp.DomainRecords.Record {
		if strings.EqualFold(rec.Type, recordType) && rec.RR == subDomain {
			return rec.Value, nil
		}
	}
	return "", nil
}

func (a *AliDNS) UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, ip string) error {
	subDomain := aliRR(recordName)
	existingID, err := a.findRecordID(ctx, cred, rootDomain, subDomain, recordType)
	if err != nil {
		return err
	}
	if existingID == "" {
		_, err = a.request(ctx, cred, map[string]string{
			"Action":     "AddDomainRecord",
			"DomainName": rootDomain,
			"RR":         subDomain,
			"Type":       recordType,
			"Value":      ip,
			"TTL":        "120",
		})
		return err
	}
	_, err = a.request(ctx, cred, map[string]string{
		"Action":   "UpdateDomainRecord",
		"RecordId": existingID,
		"RR":       subDomain,
		"Type":     recordType,
		"Value":    ip,
		"TTL":      "120",
	})
	return err
}

func (a *AliDNS) findRecordID(ctx context.Context, cred Credentials, rootDomain, subDomain, recordType string) (string, error) {
	body, err := a.request(ctx, cred, map[string]string{
		"Action":     "DescribeDomainRecords",
		"DomainName": rootDomain,
		"RRKeyWord":  subDomain,
		"Type":       recordType,
		"PageNumber": "1",
		"PageSize":   "50",
	})
	if err != nil {
		return "", err
	}
	var resp alidnsRecordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	for _, rec := range resp.DomainRecords.Record {
		if strings.EqualFold(rec.Type, recordType) && rec.RR == subDomain {
			return rec.RecordID, nil
		}
	}
	return "", nil
}

func aliRR(recordName string) string {
	subDomain := subDomainForRecord(recordName)
	if subDomain == "@" {
		return ""
	}
	return subDomain
}

func (a *AliDNS) request(ctx context.Context, cred Credentials, actionParams map[string]string) ([]byte, error) {
	if cred.Token == "" || cred.Secret == "" {
		return nil, fmt.Errorf("阿里云 DNS 凭证未配置")
	}
	query := map[string]string{
		"Format":           "JSON",
		"Version":          "2015-01-09",
		"AccessKeyId":      cred.Token,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	for k, v := range actionParams {
		query[k] = v
	}
	query["Signature"] = signAliyun(query, cred.Secret)

	values := url.Values{}
	for k, v := range query {
		values.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, alidnsEndpoint+"?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var envelope alidnsEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Code != "" {
		return nil, fmt.Errorf("阿里云 DNS API 错误：%s", envelope.Message)
	}
	return body, nil
}

func signAliyun(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "Signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var canonical strings.Builder
	for i, k := range keys {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(percentEncode(k))
		canonical.WriteByte('=')
		canonical.WriteString(percentEncode(params[k]))
	}
	stringToSign := "GET&%2F&" + percentEncode(canonical.String())
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

type alidnsEnvelope struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type alidnsRecordsResponse struct {
	DomainRecords struct {
		Record []struct {
			RecordID string `json:"RecordId"`
			RR       string `json:"RR"`
			Type     string `json:"Type"`
			Value    string `json:"Value"`
		} `json:"Record"`
	} `json:"DomainRecords"`
}
