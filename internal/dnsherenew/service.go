// Package dnsherenew 提供多个 DNSHE 账户的免费子域名管理与自动续期能力。
// 每个账户独立保存加密凭据，支持账户拖拽排序、单域名自动续期开关，
// 以及域名的删除、转赠（生成码）、接收（输入码）。
package dnsherenew

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/ddns"
	"github.com/fonu/fonu/internal/secret"
	"github.com/fonu/fonu/internal/settings"
)

// ErrAccountNotFound 表示账户不存在。
var ErrAccountNotFound = errors.New("DNSHE 账户不存在")

// ErrNoAccounts 表示尚未添加任何账户。
var ErrNoAccounts = errors.New("请先添加 DNSHE 账户")

// 官方规则：域名到期前 180 天起可续期，每次续期延长 1 年。
const renewWindowDays = 180

// 无待续期域名时，每周重新拉取一次列表以发现新注册/接收的域名。
const relistInterval = 7 * 24 * time.Hour

type Service struct {
	settings  *settings.Store
	secretBox *secret.Box
	client    *ddns.DNSHE
	logger    *slog.Logger

	mu sync.Mutex // 串行化账户读写与续期执行
}

func NewService(settingsStore *settings.Store, secretBox *secret.Box, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		settings:  settingsStore,
		secretBox: secretBox,
		client:    ddns.NewDNSHE(),
		logger:    logger.With("module", "DNSHE_RENEW"),
	}
}

// ---- 数据结构 ----

// accountRecord 是账户的持久化形态（凭据加密内嵌）。
type accountRecord struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	APIKeyEnc    string `json:"api_key_enc"`
	APISecretEnc string `json:"api_secret_enc"`
	SortOrder    int    `json:"sort_order"`
	AutoRenew    bool   `json:"auto_renew"`
	LastRunAt    string `json:"last_run_at,omitempty"`
	LastRunInfo  string `json:"last_run_info,omitempty"`
	NextCheckAt  string `json:"next_check_at,omitempty"`
}

// Account 是返回给前端的账户视图（不含凭据）。
type Account struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AutoRenew   bool   `json:"auto_renew"`
	LastRunAt   string `json:"last_run_at,omitempty"`
	LastRunInfo string `json:"last_run_info,omitempty"`
	NextCheckAt string `json:"next_check_at,omitempty"`
}

// Domain 是返回给前端的域名视图。
type Domain struct {
	ID           int64  `json:"id"`
	FullDomain   string `json:"full_domain"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	ExpiresAt    string `json:"expires_at"`
	NeverExpires bool   `json:"never_expires"`
	Renewable    bool   `json:"renewable"`
	AutoRenew    bool   `json:"auto_renew"`
	DaysLeft     *int   `json:"days_left,omitempty"`
}

type AccountInput struct {
	Name      string
	APIKey    string
	APISecret string
	AutoRenew bool
}

// ---- 账户持久化 ----

func (s *Service) loadRecords(ctx context.Context) ([]accountRecord, error) {
	raw, err := s.settings.Get(ctx, settings.KeyDNSHEAccounts)
	if err != nil {
		return nil, err
	}
	var records []accountRecord
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &records); err != nil {
			return nil, err
		}
	}
	return records, nil
}

func (s *Service) saveRecords(ctx context.Context, records []accountRecord) error {
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	return s.settings.Set(ctx, settings.KeyDNSHEAccounts, string(raw))
}

// migrateOnce 将旧版单账户配置迁移为多账户结构（仅一次）。
func (s *Service) migrateOnce(ctx context.Context, records []accountRecord) ([]accountRecord, error) {
	if len(records) > 0 {
		return records, nil
	}
	encKey, _ := s.settings.Get(ctx, settings.KeyDNSHEAPIKey)
	encSecret, _ := s.settings.Get(ctx, settings.KeyDNSHEAPISecret)
	if strings.TrimSpace(encKey) == "" && strings.TrimSpace(encSecret) == "" {
		return records, nil
	}
	autoRenew, _ := s.settings.GetBool(ctx, settings.KeyDNSHEAutoRenew)
	records = append(records, accountRecord{
		ID:           newAccountID(),
		Name:         "默认账户",
		APIKeyEnc:    encKey,
		APISecretEnc: encSecret,
		AutoRenew:    autoRenew,
	})
	if err := s.saveRecords(ctx, records); err != nil {
		return nil, err
	}
	s.logger.Info("legacy single account migrated to multi-account structure")
	return records, nil
}

func newAccountID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "a" + hex.EncodeToString(b)
}

// ---- 账户 CRUD ----

// ListAccounts 列出全部账户（自动执行旧配置迁移）。
func (s *Service) ListAccounts(ctx context.Context) ([]Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadRecords(ctx)
	if err != nil {
		return nil, err
	}
	records, err = s.migrateOnce(ctx, records)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(records))
	for _, r := range records {
		out = append(out, accountView(r))
	}
	return out, nil
}

func accountView(r accountRecord) Account {
	return Account{
		ID: r.ID, Name: r.Name, AutoRenew: r.AutoRenew,
		LastRunAt: r.LastRunAt, LastRunInfo: r.LastRunInfo, NextCheckAt: r.NextCheckAt,
	}
}

// CreateAccount 添加账户并立即验证凭据有效性。
func (s *Service) CreateAccount(ctx context.Context, in AccountInput) (*Account, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "DNSHE 账户"
	}
	apiKey := strings.TrimSpace(in.APIKey)
	apiSecret := strings.TrimSpace(in.APISecret)
	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("请填写 API Key 和 API Secret")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadRecords(ctx)
	if err != nil {
		return nil, err
	}
	records, err = s.migrateOnce(ctx, records)
	if err != nil {
		return nil, err
	}

	rec := accountRecord{
		ID: newAccountID(), Name: name, AutoRenew: in.AutoRenew,
		SortOrder: len(records),
	}
	if rec.APIKeyEnc, err = s.secretBox.Encrypt(apiKey); err != nil {
		return nil, err
	}
	if rec.APISecretEnc, err = s.secretBox.Encrypt(apiSecret); err != nil {
		return nil, err
	}
	// 保存前先验证凭据
	cred := ddns.Credentials{Provider: "dnshe", Token: apiKey, Secret: apiSecret}
	if err := s.client.Verify(ctx, cred); err != nil {
		return nil, fmt.Errorf("凭据验证失败：%w", err)
	}

	records = append(records, rec)
	if err := s.saveRecords(ctx, records); err != nil {
		return nil, err
	}
	v := accountView(rec)
	return &v, nil
}

// UpdateAccount 更新账户名称/凭据/开关；Key/Secret 留空保持不变。
func (s *Service) UpdateAccount(ctx context.Context, id string, in AccountInput) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadRecords(ctx)
	if err != nil {
		return nil, err
	}
	idx := indexOfAccount(records, id)
	if idx < 0 {
		return nil, ErrAccountNotFound
	}
	rec := &records[idx]
	if name := strings.TrimSpace(in.Name); name != "" {
		rec.Name = name
	}
	rec.AutoRenew = in.AutoRenew

	var testCred *ddns.Credentials
	if key := strings.TrimSpace(in.APIKey); key != "" {
		enc, err := s.secretBox.Encrypt(key)
		if err != nil {
			return nil, err
		}
		rec.APIKeyEnc = enc
	}
	if secretVal := strings.TrimSpace(in.APISecret); secretVal != "" {
		enc, err := s.secretBox.Encrypt(secretVal)
		if err != nil {
			return nil, err
		}
		rec.APISecretEnc = enc
	}
	if strings.TrimSpace(in.APIKey) != "" || strings.TrimSpace(in.APISecret) != "" {
		cred, err := s.accountCredentials(ctx, *rec)
		if err != nil {
			return nil, err
		}
		testCred = &cred
	}
	if testCred != nil {
		if err := s.client.Verify(ctx, *testCred); err != nil {
			return nil, fmt.Errorf("凭据验证失败：%w", err)
		}
	}
	// 配置变更后重置调度门，下一轮按新状态排期
	rec.NextCheckAt = ""

	if err := s.saveRecords(ctx, records); err != nil {
		return nil, err
	}
	v := accountView(*rec)
	return &v, nil
}

// DeleteAccount 删除账户（不影响 DNSHE 线上域名）。
func (s *Service) DeleteAccount(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadRecords(ctx)
	if err != nil {
		return err
	}
	idx := indexOfAccount(records, id)
	if idx < 0 {
		return ErrAccountNotFound
	}
	records = append(records[:idx], records[idx+1:]...)
	return s.saveRecords(ctx, records)
}

// ReorderAccounts 按给定 ID 顺序重排账户。
func (s *Service) ReorderAccounts(ctx context.Context, orderedIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadRecords(ctx)
	if err != nil {
		return err
	}
	byID := map[string]accountRecord{}
	for _, r := range records {
		byID[r.ID] = r
	}
	out := make([]accountRecord, 0, len(records))
	for i, id := range orderedIDs {
		r, ok := byID[id]
		if !ok {
			continue
		}
		r.SortOrder = i
		out = append(out, r)
	}
	if len(out) != len(records) {
		return fmt.Errorf("排序数据不完整")
	}
	return s.saveRecords(ctx, out)
}

func indexOfAccount(records []accountRecord, id string) int {
	for i := range records {
		if records[i].ID == id {
			return i
		}
	}
	return -1
}

// accountCredentials 解密账户凭据。
func (s *Service) accountCredentials(ctx context.Context, rec accountRecord) (ddns.Credentials, error) {
	if rec.APIKeyEnc == "" || rec.APISecretEnc == "" {
		return ddns.Credentials{}, fmt.Errorf("账户 %s 的凭据不完整", rec.Name)
	}
	key, err := s.secretBox.Decrypt(rec.APIKeyEnc)
	if err != nil {
		return ddns.Credentials{}, secret.DecryptHint(err)
	}
	secretVal, err := s.secretBox.Decrypt(rec.APISecretEnc)
	if err != nil {
		return ddns.Credentials{}, secret.DecryptHint(err)
	}
	return ddns.Credentials{Provider: "dnshe", Token: key, Secret: secretVal}, nil
}

func (s *Service) accountAndCredentials(ctx context.Context, id string) (*accountRecord, ddns.Credentials, error) {
	records, err := s.loadRecords(ctx)
	if err != nil {
		return nil, ddns.Credentials{}, err
	}
	idx := indexOfAccount(records, id)
	if idx < 0 {
		return nil, ddns.Credentials{}, ErrAccountNotFound
	}
	rec := records[idx]
	cred, err := s.accountCredentials(ctx, rec)
	if err != nil {
		return nil, ddns.Credentials{}, err
	}
	return &rec, cred, nil
}

// RevealCredentials 解密并返回账户的 API 凭据，供用户主动查看或推送到 DDNS。
func (s *Service) RevealCredentials(ctx context.Context, id string) (ddns.Credentials, error) {
	_, cred, err := s.accountAndCredentials(ctx, id)
	return cred, err
}

// ---- 按域名的自动续期开关 ----

func (s *Service) domainAutoRenewMap(ctx context.Context) map[string]bool {
	raw, _ := s.settings.Get(ctx, settings.KeyDNSHEDomainAutoRenew)
	m := map[string]bool{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &m)
	}
	return m
}

// SetDomainAutoRenew 设置单个域名的自动续期开关。
func (s *Service) SetDomainAutoRenew(ctx context.Context, fullDomain string, enabled bool) error {
	fullDomain = strings.ToLower(strings.TrimSpace(fullDomain))
	if fullDomain == "" {
		return fmt.Errorf("域名不能为空")
	}
	m := s.domainAutoRenewMap(ctx)
	m[fullDomain] = enabled
	raw, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.settings.Set(ctx, settings.KeyDNSHEDomainAutoRenew, string(raw))
}

// ---- 域名列表与操作 ----

func (s *Service) ListDomains(ctx context.Context, accountID string) ([]Domain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, cred, err := s.accountAndCredentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	subs, err := s.client.ListSubdomains(ctx, cred)
	if err != nil {
		return nil, err
	}
	subs = orderSubdomains(subs, s.domainOrder(ctx)[accountID])
	return decorateDomains(subs, s.domainAutoRenewMap(ctx)), nil
}

// ---- 域名排序 ----

func (s *Service) domainOrder(ctx context.Context) map[string][]int64 {
	raw, _ := s.settings.Get(ctx, settings.KeyDNSHEDomainOrder)
	m := map[string][]int64{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &m)
	}
	return m
}

// orderSubdomains 按持久化顺序排列；已删除的 ID 被忽略，新域名追加末尾。
func orderSubdomains(subs []ddns.Subdomain, order []int64) []ddns.Subdomain {
	if len(order) == 0 {
		return subs
	}
	byID := make(map[int64]ddns.Subdomain, len(subs))
	for _, sub := range subs {
		byID[sub.ID] = sub
	}
	out := make([]ddns.Subdomain, 0, len(subs))
	seen := map[int64]bool{}
	for _, id := range order {
		if sub, ok := byID[id]; ok && !seen[id] {
			out = append(out, sub)
			seen[id] = true
		}
	}
	for _, sub := range subs {
		if !seen[sub.ID] {
			out = append(out, sub)
		}
	}
	return out
}

// ReorderDomains 保存账户下的域名显示顺序。
func (s *Service) ReorderDomains(ctx context.Context, accountID string, ids []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	order := s.domainOrder(ctx)
	order[accountID] = ids
	raw, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return s.settings.Set(ctx, settings.KeyDNSHEDomainOrder, string(raw))
}

// appendDomainOrder 将新注册域名追加到顺序末尾。
func (s *Service) appendDomainOrder(ctx context.Context, accountID string, id int64) error {
	order := s.domainOrder(ctx)
	for _, x := range order[accountID] {
		if x == id {
			return nil
		}
	}
	order[accountID] = append(order[accountID], id)
	raw, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return s.settings.Set(ctx, settings.KeyDNSHEDomainOrder, string(raw))
}

// RegisterDomain 注册新的免费域名。
func (s *Service) RegisterDomain(ctx context.Context, accountID, prefix, rootDomain string) (*ddns.RegisterResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, cred, err := s.accountAndCredentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	result, err := s.client.RegisterSubdomain(ctx, cred, prefix, rootDomain)
	if err != nil {
		return nil, err
	}
	if err := s.appendDomainOrder(ctx, accountID, result.SubdomainID); err != nil {
		s.logger.Warn("append domain order failed", "error", err)
	}
	return result, nil
}

func decorateDomains(subs []ddns.Subdomain, autoMap map[string]bool) []Domain {
	out := make([]Domain, 0, len(subs))
	for _, sub := range subs {
		autoRenew, exists := autoMap[strings.ToLower(sub.FullDomain)]
		if !exists {
			autoRenew = true
		}
		d := Domain{
			ID: sub.ID, FullDomain: sub.FullDomain, Status: sub.Status,
			CreatedAt: sub.CreatedAt, ExpiresAt: sub.ExpiresAt,
			NeverExpires: sub.NeverExpires, Renewable: !sub.NeverExpires, AutoRenew: autoRenew,
		}
		if !sub.NeverExpires {
			if exp, ok := parseDNSHETime(sub.ExpiresAt); ok {
				days := int(time.Until(exp).Hours() / 24)
				d.DaysLeft = &days
			}
		}
		out = append(out, d)
	}
	return out
}

// RenewDomain 手动续期单个域名。
func (s *Service) RenewDomain(ctx context.Context, accountID string, subdomainID int64) (*ddns.RenewResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, cred, err := s.accountAndCredentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.client.RenewSubdomain(ctx, cred, subdomainID)
}

// DeleteDomain 删除单个域名。
func (s *Service) DeleteDomain(ctx context.Context, accountID string, subdomainID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, cred, err := s.accountAndCredentials(ctx, accountID)
	if err != nil {
		return err
	}
	return s.client.DeleteSubdomain(ctx, cred, subdomainID)
}

// InitiateGift 为域名生成转赠码。
func (s *Service) InitiateGift(ctx context.Context, accountID string, subdomainID int64) (*ddns.Gift, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, cred, err := s.accountAndCredentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.client.InitiateGift(ctx, cred, subdomainID)
}

// AcceptGift 凭转赠码接收域名。
func (s *Service) AcceptGift(ctx context.Context, accountID, code string) (*ddns.Gift, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, cred, err := s.accountAndCredentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.client.AcceptGift(ctx, cred, code)
}

// CancelGift 取消尚未被接收的转赠。
func (s *Service) CancelGift(ctx context.Context, accountID string, giftID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, cred, err := s.accountAndCredentials(ctx, accountID)
	if err != nil {
		return err
	}
	return s.client.CancelGift(ctx, cred, giftID)
}

// ListGifts 列出账户的转赠记录。
func (s *Service) ListGifts(ctx context.Context, accountID string) ([]ddns.Gift, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, cred, err := s.accountAndCredentials(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.client.ListGifts(ctx, cred)
}

// ---- 自动续期调度 ----

// Tick 是调度器入口（底层 ticker 每天唤醒一次，绝大多数轮次零 API 调用）。
// 逐账户检查：未开启自动续期或未到该账户的「下次检查时间」则跳过。
func (s *Service) Tick(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadRecords(ctx)
	if err != nil {
		return
	}
	records, err = s.migrateOnce(ctx, records)
	if err != nil {
		return
	}
	if len(records) == 0 {
		return
	}

	now := time.Now()
	for i := range records {
		rec := &records[i]
		if !rec.AutoRenew {
			continue
		}
		if rec.NextCheckAt != "" {
			if next, ok := parseDNSHETime(rec.NextCheckAt); ok && now.Before(next) {
				continue // 未到下次检查时间
			}
		}
		s.tickAccount(ctx, rec, now)
	}
	_ = s.saveRecords(ctx, records)
}

func (s *Service) tickAccount(ctx context.Context, rec *accountRecord, now time.Time) {
	cred, err := s.accountCredentials(ctx, *rec)
	if err != nil {
		rec.LastRunInfo = "凭据不可用：" + err.Error()
		rec.NextCheckAt = now.Add(24 * time.Hour).Format("2006-01-02 15:04:05")
		return
	}

	subs, err := s.client.ListSubdomains(ctx, cred)
	if err != nil {
		s.logger.Warn("auto renew: list domains failed", "account", rec.Name, "error", err)
		rec.LastRunInfo = fmt.Sprintf("获取域名列表失败：%v", err)
		rec.NextCheckAt = now.Add(24 * time.Hour).Format("2006-01-02 15:04:05")
		return
	}

	renewed, failed := 0, 0
	nextCheck := now.Add(relistInterval)
	tryTomorrow := func() {
		if t := now.Add(24 * time.Hour); t.Before(nextCheck) {
			nextCheck = t
		}
	}
	autoMap := s.domainAutoRenewMap(ctx)
	for _, sub := range subs {
		if sub.NeverExpires {
			continue
		}
		if enabled, exists := autoMap[strings.ToLower(sub.FullDomain)]; exists && !enabled {
			continue // 该域名已关闭自动续期
		}
		exp, ok := parseDNSHETime(sub.ExpiresAt)
		if !ok {
			tryTomorrow()
			continue
		}
		if openAt := exp.AddDate(0, 0, -renewWindowDays); now.Before(openAt) {
			if openAt.Before(nextCheck) {
				nextCheck = openAt
			}
			continue
		}
		result, err := s.client.RenewSubdomain(ctx, cred, sub.ID)
		if err != nil {
			if !strings.Contains(err.Error(), "renewal_not_yet_available") {
				failed++
				s.logger.Warn("auto renew failed", "account", rec.Name, "domain", sub.FullDomain, "error", err)
			}
			tryTomorrow()
			continue
		}
		renewed++
		s.logger.Info("domain renewed", "account", rec.Name, "domain", sub.FullDomain, "new_expires_at", result.NewExpiresAt)
		if newExp, ok2 := parseDNSHETime(result.NewExpiresAt); ok2 {
			if o := newExp.AddDate(0, 0, -renewWindowDays); o.Before(nextCheck) {
				nextCheck = o
			}
		}
	}

	loc := s.settings.Location(ctx)
	rec.LastRunAt = now.In(loc).Format("2006-01-02 15:04:05")
	rec.LastRunInfo = fmt.Sprintf("续期成功 %d 个，失败 %d 个", renewed, failed)
	rec.NextCheckAt = nextCheck.In(loc).Format("2006-01-02 15:04:05")
}

// parseDNSHETime 兼容 DNSHE 返回的多种时间格式。
func parseDNSHETime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	for _, f := range []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(f, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
