package ddns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DNSHE 免费域名（https://my.dnshe.com/）
// API 文档：https://my.dnshe.com/knowledgebase/13/DNSHE免费域名API使用文档V2.0.html
// 认证方式：X-API-Key + X-API-Secret 请求头
const (
	dnsheBaseURL    = "https://api005.dnshe.com/index.php"
	dnsheDefaultTTL = 600
	// DNSHE 子域名列表较短（免费账户通常只有几个），短时缓存可避免重复请求。
	dnsheZoneCacheTTL = 5 * time.Minute
)

type DNSHE struct {
	client *http.Client

	cacheMu sync.Mutex
	cache   map[string]dnsheZoneCache
}

type dnsheZoneCache struct {
	expires time.Time
	subs    []dnsheSubdomain
}

func NewDNSHE() *DNSHE {
	return &DNSHE{
		client: &http.Client{Timeout: 30 * time.Second},
		cache:  map[string]dnsheZoneCache{},
	}
}

func (d *DNSHE) Name() string { return "dnshe" }

// Verify 通过拉取子域名列表验证 API Key / Secret 是否有效。
func (d *DNSHE) Verify(ctx context.Context, cred Credentials) error {
	_, err := d.listSubdomains(ctx, cred, 1, 1)
	return err
}

// HasZone 判断免费子域名（如 example.cc.cd）是否属于当前账户。
func (d *DNSHE) HasZone(ctx context.Context, cred Credentials, zone string) (bool, error) {
	id, err := d.subdomainID(ctx, cred, zone)
	if err != nil {
		return false, err
	}
	return id > 0, nil
}

func (d *DNSHE) GetRecordIP(ctx context.Context, cred Credentials, rootDomain, recordName, recordType string) (string, error) {
	subID, err := d.subdomainID(ctx, cred, rootDomain)
	if err != nil {
		return "", err
	}
	fqdn := dnsheFQDN(rootDomain, recordName)
	rec, err := d.findRecord(ctx, cred, subID, fqdn, recordType)
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", nil
	}
	return rec.Content, nil
}

func (d *DNSHE) UpdateRecord(ctx context.Context, cred Credentials, rootDomain, recordName, recordType, ip string) error {
	subID, err := d.subdomainID(ctx, cred, rootDomain)
	if err != nil {
		return err
	}
	fqdn := dnsheFQDN(rootDomain, recordName)
	return d.upsertRecord(ctx, cred, subID, rootDomain, fqdn, recordType, ip)
}

func (d *DNSHE) upsertRecord(ctx context.Context, cred Credentials, subID int64, zoneFQDN, fqdn, recordType, content string) error {
	rec, err := d.findRecord(ctx, cred, subID, fqdn, recordType)
	if err != nil {
		return err
	}
	if rec != nil {
		payload := map[string]any{
			"id":      rec.ID,
			"type":    strings.ToUpper(recordType),
			"content": content,
			"ttl":     dnsheDefaultTTL,
		}
		var resp dnsheMessageResponse
		return d.call(ctx, cred, http.MethodPost, "dns_records", "update", nil, payload, &resp)
	}
	payload := map[string]any{
		"subdomain_id": subID,
		"type":         strings.ToUpper(recordType),
		"content":      content,
		"ttl":          dnsheDefaultTTL,
	}
	// name 留空或 "@" 表示子域名本身，其余支持填写完整域名。
	if name := dnsheRecordNameParam(zoneFQDN, fqdn); name != "" {
		payload["name"] = name
	}
	var resp dnsheMessageResponse
	return d.call(ctx, cred, http.MethodPost, "dns_records", "create", nil, payload, &resp)
}

// ensureTXTRecord 确保存在一条内容为 value 的 TXT 记录。
// 证书同时覆盖根域与泛域名时，Let's Encrypt 会要求同名 TXT 同时存在多个不同值，
// 因此按“名称+类型+内容”匹配，已存在则跳过，不存在才新建，避免互相覆盖。
func (d *DNSHE) ensureTXTRecord(ctx context.Context, cred Credentials, subID int64, zoneFQDN, fqdn, value string) error {
	records, err := d.listRecords(ctx, cred, subID)
	if err != nil {
		return err
	}
	for _, rec := range records {
		name := strings.ToLower(strings.TrimSuffix(rec.Name, "."))
		fqdnLower := strings.ToLower(strings.TrimSuffix(fqdn, "."))
		if name != fqdnLower && !dnsheNameMatchesFQDN(name, fqdnLower) {
			continue
		}
		if !strings.EqualFold(rec.Type, "TXT") || strings.TrimSpace(rec.Content) != strings.TrimSpace(value) {
			continue
		}
		return nil
	}
	payload := map[string]any{
		"subdomain_id": subID,
		"type":         "TXT",
		"content":      value,
		"ttl":          dnsheDefaultTTL,
	}
	if name := dnsheRecordNameParam(zoneFQDN, fqdn); name != "" {
		payload["name"] = name
	}
	var resp dnsheMessageResponse
	return d.call(ctx, cred, http.MethodPost, "dns_records", "create", nil, payload, &resp)
}

func (d *DNSHE) deleteRecords(ctx context.Context, cred Credentials, subID int64, fqdn, recordType, content string) error {
	records, err := d.listRecords(ctx, cred, subID)
	if err != nil {
		return err
	}
	fqdnLower := strings.ToLower(strings.TrimSuffix(fqdn, "."))
	for _, rec := range records {
		name := strings.ToLower(strings.TrimSuffix(rec.Name, "."))
		if name != fqdnLower && !dnsheNameMatchesFQDN(name, fqdnLower) {
			continue
		}
		if !strings.EqualFold(rec.Type, recordType) {
			continue
		}
		if content != "" && strings.TrimSpace(rec.Content) != strings.TrimSpace(content) {
			continue
		}
		var resp dnsheMessageResponse
		if err := d.call(ctx, cred, http.MethodPost, "dns_records", "delete", nil, map[string]any{
			"id": rec.ID,
		}, &resp); err != nil {
			return err
		}
	}
	return nil
}

func (d *DNSHE) findRecord(ctx context.Context, cred Credentials, subID int64, fqdn, recordType string) (*dnsheRecord, error) {
	records, err := d.listRecords(ctx, cred, subID)
	if err != nil {
		return nil, err
	}
	fqdn = strings.ToLower(strings.TrimSuffix(fqdn, "."))
	for i := range records {
		rec := &records[i]
		name := strings.ToLower(strings.TrimSuffix(rec.Name, "."))
		if (name == fqdn || dnsheNameMatchesFQDN(name, fqdn)) && strings.EqualFold(rec.Type, recordType) {
			return rec, nil
		}
	}
	return nil, nil
}

// dnsheNameMatchesFQDN 判断 API 返回的记录名是否与目标 FQDN 匹配。
// API 可能返回完整域名（_acme-challenge.example.com）或相对名（_acme-challenge）。
func dnsheNameMatchesFQDN(apiName, fqdn string) bool {
	if apiName == fqdn {
		return true
	}
	// apiName 是相对名时，检查后缀
	parts := strings.SplitN(fqdn, ".", 2)
	if len(parts) == 2 && parts[0] == apiName {
		return true
	}
	return false
}

func (d *DNSHE) listRecords(ctx context.Context, cred Credentials, subID int64) ([]dnsheRecord, error) {
	query := url.Values{}
	query.Set("subdomain_id", fmt.Sprintf("%d", subID))
	var resp dnsheRecordsResponse
	if err := d.call(ctx, cred, http.MethodGet, "dns_records", "list", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Records, nil
}

// subdomainID 返回免费子域名的内部 ID；不存在时返回 0。
func (d *DNSHE) subdomainID(ctx context.Context, cred Credentials, fullDomain string) (int64, error) {
	subs, err := d.allSubdomains(ctx, cred)
	if err != nil {
		return 0, err
	}
	fullDomain = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(fullDomain), "."))
	for _, sub := range subs {
		name := strings.TrimSuffix(sub.FullDomain, ".")
		if name == "" && sub.Subdomain != "" && sub.RootDomain != "" {
			name = sub.Subdomain + "." + sub.RootDomain
		}
		if strings.EqualFold(name, fullDomain) {
			return sub.ID, nil
		}
	}
	return 0, nil
}

func (d *DNSHE) allSubdomains(ctx context.Context, cred Credentials) ([]dnsheSubdomain, error) {
	if cred.Token == "" || cred.Secret == "" {
		return nil, fmt.Errorf("DNSHE API Key / Secret 未配置")
	}

	d.cacheMu.Lock()
	if entry, ok := d.cache[cred.Token]; ok && time.Now().Before(entry.expires) {
		d.cacheMu.Unlock()
		return entry.subs, nil
	}
	d.cacheMu.Unlock()

	var all []dnsheSubdomain
	page := 1
	for {
		subs, err := d.listSubdomains(ctx, cred, page, 500)
		if err != nil {
			return nil, err
		}
		all = append(all, subs...)
		if len(subs) < 500 {
			break
		}
		page++
	}

	d.cacheMu.Lock()
	d.cache[cred.Token] = dnsheZoneCache{expires: time.Now().Add(dnsheZoneCacheTTL), subs: all}
	d.cacheMu.Unlock()
	return all, nil
}

func (d *DNSHE) listSubdomains(ctx context.Context, cred Credentials, page, perPage int) ([]dnsheSubdomain, error) {
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	var resp dnsheSubdomainsResponse
	if err := d.call(ctx, cred, http.MethodGet, "subdomains", "list", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Subdomains, nil
}

// call 调用 DNSHE API 并解析统一响应信封。
func (d *DNSHE) call(ctx context.Context, cred Credentials, method, endpoint, action string, query url.Values, payload any, out any) error {
	if cred.Token == "" || cred.Secret == "" {
		return fmt.Errorf("DNSHE API Key / Secret 未配置")
	}

	apiURL := dnsheBaseURL + "?m=domain_hub&endpoint=" + url.PathEscape(endpoint) + "&action=" + url.PathEscape(action)
	if len(query) > 0 {
		apiURL += "&" + query.Encode()
	}

	// DNSHE API（经 Cloudflare）偶发 TLS 握手/响应超时：GET 类请求网络层失败时最多重试 2 次
	maxAttempts := 1
	if method == http.MethodGet {
		maxAttempts = 3
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}

		var attemptReader io.Reader
		if payload != nil && method != http.MethodGet {
			raw, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			attemptReader = bytes.NewReader(raw)
		}
		req, err := http.NewRequestWithContext(ctx, method, apiURL, attemptReader)
		if err != nil {
			return err
		}
		req.Header.Set("X-API-Key", cred.Token)
		req.Header.Set("X-API-Secret", cred.Secret)
		req.Header.Set("Accept", "application/json")
		if payload != nil && method != http.MethodGet {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts && isTransientNetErr(err) {
				continue
			}
			return err
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < maxAttempts && isTransientNetErr(readErr) {
				continue
			}
			return readErr
		}

		var base dnsheBase
		if err := json.Unmarshal(body, &base); err != nil {
			if resp.StatusCode >= 400 {
				return fmt.Errorf("DNSHE API 请求失败（HTTP %d）", resp.StatusCode)
			}
			return fmt.Errorf("解析 DNSHE API 响应失败：%w", err)
		}
		if resp.StatusCode >= 400 || !base.Success {
			msg := base.Message
			if msg == "" {
				msg = base.Error
			}
			if msg == "" {
				msg = fmt.Sprintf("DNSHE API 请求失败（HTTP %d）", resp.StatusCode)
			}
			// 429/5xx 视为临时故障，可重试
			if attempt < maxAttempts && (resp.StatusCode == 429 || resp.StatusCode >= 500) {
				lastErr = fmt.Errorf("DNSHE API 临时故障（HTTP %d）", resp.StatusCode)
				continue
			}
			if base.ErrorCode != "" {
				return fmt.Errorf("DNSHE API 错误：%s（%s）", msg, base.ErrorCode)
			}
			return fmt.Errorf("DNSHE API 错误：%s", msg)
		}
		if out != nil {
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("解析 DNSHE API 响应失败：%w", err)
			}
		}
		return nil
	}
	return lastErr
}

// isTransientNetErr 判断是否为值得重试的临时网络错误（超时、连接重置、TLS 握手超时等）。
func isTransientNetErr(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := err.Error()
	for _, marker := range []string{
		"TLS handshake timeout",
		"context deadline exceeded",
		"connection reset by peer",
		"EOF",
		"broken pipe",
		"no such host",
		"i/o timeout",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// dnsheFQDN 根据根域名（免费子域名）与记录名构造完整域名。
// recordName 支持 "@"、单级主机名（nas）、多级相对名（a.b）或已是完整域名。
func dnsheFQDN(rootDomain, recordName string) string {
	root := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(rootDomain), "."))
	name := strings.TrimSpace(recordName)
	if name == "" || name == "@" {
		return root
	}
	name = strings.TrimSuffix(name, ".")
	if strings.HasSuffix(strings.ToLower(name), "."+root) {
		return name
	}
	return name + "." + root
}

// dnsheRecordNameParam 返回创建记录时使用的 name 参数：子域名根记录留空，其余返回相对记录名。
// DNSHE API 实际要求相对名（如 "_acme-challenge"），不接受完整域名。
func dnsheRecordNameParam(zoneFQDN, fqdn string) string {
	zone := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(zoneFQDN), "."))
	full := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(fqdn), "."))
	if zone == full {
		return ""
	}
	if strings.HasSuffix(full, "."+zone) {
		return full[:len(full)-len(zone)-1]
	}
	return fqdn
}

type dnsheBase struct {
	Success   bool   `json:"success"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	Error     string `json:"error"`
}

type dnsheSubdomainsResponse struct {
	dnsheBase
	Count      int `json:"count"`
	Subdomains []dnsheSubdomain `json:"subdomains"`
	Pagination struct {
		HasMore  bool `json:"has_more"`
		NextPage int  `json:"next_page"`
	} `json:"pagination"`
}

type dnsheRecordsResponse struct {
	dnsheBase
	Count   int           `json:"count"`
	Records []dnsheRecord `json:"records"`
}

type dnsheMessageResponse struct {
	dnsheBase
	ID       int64  `json:"id"`
	RecordID string `json:"record_id"`
}

type dnsheSubdomain struct {
	ID           int64         `json:"id"`
	Subdomain    string        `json:"subdomain"`
	RootDomain   string        `json:"rootdomain"`
	FullDomain   string        `json:"full_domain"`
	Status       string        `json:"status"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	ExpiresAt    string        `json:"expires_at"`
	NeverExpires dnsheFlexBool `json:"never_expires"`
}

// dnsheFlexBool 兼容 API 可能返回的 bool / 数字 / 字符串形式的布尔值。
type dnsheFlexBool bool

func (b *dnsheFlexBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(strings.TrimSpace(string(data)), `"`)
	*b = s == "true" || s == "1" || s == "yes"
	return nil
}

type dnsheRecord struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Status  string `json:"status"`
}
