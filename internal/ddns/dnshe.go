package ddns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const dnsheAPIBase = "https://api005.dnshe.com/index.php?m=domain_hub"

type DNSHE struct {
	client  *http.Client
	baseURL string
}

func NewDNSHE() *DNSHE {
	return &DNSHE{client: &http.Client{Timeout: 30 * time.Second}}
}

func (d *DNSHE) apiBase() string {
	if strings.TrimSpace(d.baseURL) != "" {
		return d.baseURL
	}
	return dnsheAPIBase
}

func (d *DNSHE) Name() string { return "dnshe" }

func (d *DNSHE) Verify(ctx context.Context, cred Credentials) error {
	if err := cred.Validate("dnshe"); err != nil {
		return err
	}
	_, _, err := d.listSubdomainsPage(ctx, cred, 1, 1, "")
	return err
}

func (d *DNSHE) HasZone(ctx context.Context, cred Credentials, zone string) (bool, error) {
	_, err := d.subdomainForZone(ctx, cred, zone)
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "未找到") {
		return false, nil
	}
	return false, err
}

func (d *DNSHE) GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error) {
	fqdn := normalizeDNSHEHost(fqdnForRecord(rootDomain, recordName))
	sub, err := d.subdomainForFQDN(ctx, cred, fqdn)
	if err != nil {
		return "", err
	}
	records, err := d.listDNSRecords(ctx, cred, sub.ID)
	if err != nil {
		return "", err
	}
	for _, rec := range records {
		if !strings.EqualFold(rec.Type, recordType) {
			continue
		}
		if dnsheRecordMatches(rec.Name, sub.FullDomain, fqdn) {
			return strings.TrimSpace(rec.Content), nil
		}
	}
	return "", nil
}

func (d *DNSHE) UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, value string) error {
	fqdn := normalizeDNSHEHost(fqdnForRecord(rootDomain, recordName))
	sub, err := d.subdomainForFQDN(ctx, cred, fqdn)
	if err != nil {
		return err
	}
	name := dnsheAPIRecordName(sub.FullDomain, fqdn)
	records, err := d.listDNSRecords(ctx, cred, sub.ID)
	if err != nil {
		return err
	}
	for _, rec := range records {
		if !strings.EqualFold(rec.Type, recordType) {
			continue
		}
		if !dnsheRecordMatches(rec.Name, sub.FullDomain, fqdn) {
			continue
		}
		return d.updateDNSRecord(ctx, cred, rec.ID, recordType, name, value, rec.TTL)
	}
	return d.createDNSRecord(ctx, cred, sub.ID, recordType, name, value, 600)
}

func (d *DNSHE) UpsertTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	fqdn = normalizeDNSHEHost(fqdn)
	zone, host, err := ResolveZoneForFQDN(ctx, func(candidate string) (bool, error) {
		return d.HasZone(ctx, cred, candidate)
	}, fqdn)
	if err != nil {
		return err
	}
	return d.UpdateRecord(ctx, cred, zone, host, "TXT", value)
}

func (d *DNSHE) DeleteTXTRecord(ctx context.Context, cred Credentials, fqdn, value string) error {
	fqdn = normalizeDNSHEHost(fqdn)
	sub, err := d.subdomainForFQDN(ctx, cred, fqdn)
	if err != nil {
		return err
	}
	records, err := d.listDNSRecords(ctx, cred, sub.ID)
	if err != nil {
		return err
	}
	for _, rec := range records {
		if !strings.EqualFold(rec.Type, "TXT") {
			continue
		}
		if !dnsheRecordMatches(rec.Name, sub.FullDomain, fqdn) {
			continue
		}
		if value != "" && strings.TrimSpace(rec.Content) != strings.TrimSpace(value) {
			continue
		}
		return d.deleteDNSRecord(ctx, cred, rec.ID)
	}
	return nil
}

type dnsheSubdomain struct {
	ID         int
	FullDomain string
}

type dnsheDNSRecord struct {
	ID      int
	Name    string
	Type    string
	Content string
	TTL     int
}

func (d *DNSHE) subdomainForZone(ctx context.Context, cred Credentials, zone string) (dnsheSubdomain, error) {
	zone = normalizeDNSHEHost(zone)
	subs, err := d.searchSubdomains(ctx, cred, zone)
	if err != nil {
		return dnsheSubdomain{}, err
	}
	for _, s := range subs {
		if normalizeDNSHEHost(s.FullDomain) == zone {
			return s, nil
		}
	}
	return dnsheSubdomain{}, fmt.Errorf("未在 DNSHE 中找到子域名 %s", zone)
}

func (d *DNSHE) subdomainForFQDN(ctx context.Context, cred Credentials, fqdn string) (dnsheSubdomain, error) {
	fqdn = normalizeDNSHEHost(fqdn)
	var best dnsheSubdomain
	found := false
	for page := 1; page <= 100; page++ {
		subs, hasMore, err := d.listSubdomainsPage(ctx, cred, page, 200, "")
		if err != nil {
			return dnsheSubdomain{}, err
		}
		for _, s := range subs {
			apex := normalizeDNSHEHost(s.FullDomain)
			if fqdn == apex || strings.HasSuffix(fqdn, "."+apex) {
				if !found || len(apex) > len(best.FullDomain) {
					best = s
					found = true
				}
			}
		}
		if found {
			return best, nil
		}
		if !hasMore {
			break
		}
	}
	return dnsheSubdomain{}, fmt.Errorf("未在 DNSHE 中找到域名 %s 对应的子域名", fqdn)
}

func (d *DNSHE) searchSubdomains(ctx context.Context, cred Credentials, search string) ([]dnsheSubdomain, error) {
	var out []dnsheSubdomain
	for page := 1; page <= 20; page++ {
		subs, hasMore, err := d.listSubdomainsPage(ctx, cred, page, 200, search)
		if err != nil {
			return nil, err
		}
		out = append(out, subs...)
		if !hasMore {
			break
		}
	}
	return out, nil
}

func (d *DNSHE) listSubdomainsPage(ctx context.Context, cred Credentials, page, perPage int, search string) ([]dnsheSubdomain, bool, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	if search != "" {
		q.Set("search", search)
	}
	raw, err := d.request(ctx, cred, http.MethodGet, "subdomains", "list", q, nil)
	if err != nil {
		return nil, false, err
	}
	var payload struct {
		Subdomains []struct {
			ID         int    `json:"id"`
			FullDomain string `json:"full_domain"`
		} `json:"subdomains"`
		Pagination struct {
			HasMore bool `json:"has_more"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false, err
	}
	out := make([]dnsheSubdomain, 0, len(payload.Subdomains))
	for _, s := range payload.Subdomains {
		out = append(out, dnsheSubdomain{ID: s.ID, FullDomain: s.FullDomain})
	}
	return out, payload.Pagination.HasMore, nil
}

func (d *DNSHE) listDNSRecords(ctx context.Context, cred Credentials, subdomainID int) ([]dnsheDNSRecord, error) {
	q := url.Values{}
	q.Set("subdomain_id", strconv.Itoa(subdomainID))
	raw, err := d.request(ctx, cred, http.MethodGet, "dns_records", "list", q, nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Records []dnsheDNSRecord `json:"records"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload.Records, nil
}

func (d *DNSHE) createDNSRecord(ctx context.Context, cred Credentials, subdomainID int, recordType, name, content string, ttl int) error {
	body := map[string]any{
		"subdomain_id": subdomainID,
		"type":         recordType,
		"content":      content,
		"ttl":          ttl,
	}
	if name != "" && name != "@" {
		body["name"] = name
	}
	_, err := d.request(ctx, cred, http.MethodPost, "dns_records", "create", nil, body)
	return err
}

func (d *DNSHE) updateDNSRecord(ctx context.Context, cred Credentials, id int, recordType, name, content string, ttl int) error {
	body := map[string]any{
		"id":      id,
		"type":    recordType,
		"content": content,
		"ttl":     ttl,
	}
	if name != "" && name != "@" {
		body["name"] = name
	}
	_, err := d.request(ctx, cred, http.MethodPost, "dns_records", "update", nil, body)
	return err
}

func (d *DNSHE) deleteDNSRecord(ctx context.Context, cred Credentials, id int) error {
	body := map[string]any{"id": id}
	_, err := d.request(ctx, cred, http.MethodPost, "dns_records", "delete", nil, body)
	return err
}

func (d *DNSHE) request(ctx context.Context, cred Credentials, method, endpoint, action string, query url.Values, body any) (json.RawMessage, error) {
	u, err := url.Parse(d.apiBase())
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("endpoint", endpoint)
	q.Set("action", action)
	for k, vs := range query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", strings.TrimSpace(cred.Token))
	req.Header.Set("X-API-Secret", strings.TrimSpace(cred.Secret))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 DNSHE API 失败：%w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Success bool            `json:"success"`
		Error   string            `json:"error"`
		Message string            `json:"message"`
		Data    json.RawMessage   `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("DNSHE API 返回 %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("解析 DNSHE 响应失败：%w", err)
	}
	if !envelope.Success {
		msg := strings.TrimSpace(envelope.Error)
		if msg == "" {
			msg = strings.TrimSpace(envelope.Message)
		}
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("DNSHE API 错误：%s", msg)
	}
	return raw, nil
}

func normalizeDNSHEHost(host string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimSuffix(host, ".")))
}

func dnsheAPIRecordName(apex, fqdn string) string {
	apex = normalizeDNSHEHost(apex)
	fqdn = normalizeDNSHEHost(fqdn)
	if fqdn == apex {
		return "@"
	}
	if strings.HasSuffix(fqdn, "."+apex) {
		return strings.TrimSuffix(fqdn, "."+apex)
	}
	return fqdn
}

func dnsheRecordMatches(recordName, apex, fqdn string) bool {
	recordName = normalizeDNSHEHost(recordName)
	apex = normalizeDNSHEHost(apex)
	fqdn = normalizeDNSHEHost(fqdn)
	if recordName == fqdn {
		return true
	}
	if recordName == apex && fqdn == apex {
		return true
	}
	rel := dnsheAPIRecordName(apex, fqdn)
	if rel == "@" && (recordName == "" || recordName == apex) {
		return true
	}
	return recordName == rel || recordName == rel+"."+apex
}
