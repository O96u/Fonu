package ddns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Cloudflare struct {
	client *http.Client
}

func NewCloudflare() *Cloudflare {
	return &Cloudflare{client: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Cloudflare) Name() string { return "cloudflare" }

func (c *Cloudflare) Verify(ctx context.Context, cred Credentials) error {
	_, err := c.request(ctx, cred.Token, http.MethodGet, "https://api.cloudflare.com/client/v4/user/tokens/verify", nil)
	return err
}

func (c *Cloudflare) GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error) {
	zoneID, err := c.zoneID(ctx, cred.Token, rootDomain)
	if err != nil {
		return "", err
	}
	fqdn := fqdnForRecord(rootDomain, recordName)
	body, err := c.request(ctx, cred.Token, http.MethodGet, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=%s&name=%s", zoneID, recordType, fqdn), nil)
	if err != nil {
		return "", err
	}
	var resp cfRecordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if len(resp.Result) == 0 {
		return "", nil
	}
	return resp.Result[0].Content, nil
}

func (c *Cloudflare) UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, ip string) error {
	zoneID, err := c.zoneID(ctx, cred.Token, rootDomain)
	if err != nil {
		return err
	}
	fqdn := fqdnForRecord(rootDomain, recordName)
	existingID, _, err := c.findRecord(ctx, cred.Token, zoneID, recordType, fqdn)
	if err != nil {
		return err
	}
	payload := map[string]any{
		"type":    recordType,
		"name":    fqdn,
		"content": ip,
		"ttl":     120,
		"proxied": false,
	}
	if existingID == "" {
		_, err = c.request(ctx, cred.Token, http.MethodPost, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID), payload)
		return err
	}
	_, err = c.request(ctx, cred.Token, http.MethodPut, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, existingID), payload)
	return err
}

func (c *Cloudflare) zoneID(ctx context.Context, token, rootDomain string) (string, error) {
	body, err := c.request(ctx, token, http.MethodGet, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", rootDomain), nil)
	if err != nil {
		return "", err
	}
	var resp cfZonesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if len(resp.Result) == 0 {
		return "", fmt.Errorf("未找到域名 %s 对应的 Cloudflare Zone", rootDomain)
	}
	return resp.Result[0].ID, nil
}

func (c *Cloudflare) findRecord(ctx context.Context, token, zoneID, recordType, fqdn string) (string, string, error) {
	body, err := c.request(ctx, token, http.MethodGet, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=%s&name=%s", zoneID, recordType, fqdn), nil)
	if err != nil {
		return "", "", err
	}
	var resp cfRecordsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", "", err
	}
	if len(resp.Result) == 0 {
		return "", "", nil
	}
	return resp.Result[0].ID, resp.Result[0].Content, nil
}

func (c *Cloudflare) request(ctx context.Context, token, method, url string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var envelope cfEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if !envelope.Success {
		if len(envelope.Errors) > 0 {
			return nil, fmt.Errorf("Cloudflare API 错误：%s", envelope.Errors[0].Message)
		}
		return nil, fmt.Errorf("Cloudflare API 请求失败")
	}
	return raw, nil
}

type cfEnvelope struct {
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type cfZonesResponse struct {
	Result []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
}

type cfRecordsResponse struct {
	Result []struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	} `json:"result"`
}
