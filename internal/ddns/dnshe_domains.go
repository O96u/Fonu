package ddns

import (
	"context"
	"net/http"
)

// Subdomain 是 DNSHE 免费子域名的公开视图（含注册/到期时间）。
type Subdomain struct {
	ID           int64  `json:"id"`
	FullDomain   string `json:"full_domain"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	ExpiresAt    string `json:"expires_at"`
	NeverExpires bool   `json:"never_expires"`
}

// RenewResult 是 DNSHE 子域名续期结果。
type RenewResult struct {
	FullDomain        string `json:"full_domain"`
	PreviousExpiresAt string `json:"previous_expires_at"`
	NewExpiresAt      string `json:"new_expires_at"`
	NeverExpires      bool   `json:"never_expires"`
	Status            string `json:"status"`
	RemainingDays     int    `json:"remaining_days"`
}

// ListSubdomains 返回账户下全部免费子域名（带注册/到期时间，使用短时缓存）。
func (d *DNSHE) ListSubdomains(ctx context.Context, cred Credentials) ([]Subdomain, error) {
	subs, err := d.allSubdomains(ctx, cred)
	if err != nil {
		return nil, err
	}
	out := make([]Subdomain, 0, len(subs))
	for _, sub := range subs {
		full := sub.FullDomain
		if full == "" && sub.Subdomain != "" && sub.RootDomain != "" {
			full = sub.Subdomain + "." + sub.RootDomain
		}
		out = append(out, Subdomain{
			ID:           sub.ID,
			FullDomain:   full,
			Status:       sub.Status,
			CreatedAt:    sub.CreatedAt,
			ExpiresAt:    sub.ExpiresAt,
			NeverExpires: bool(sub.NeverExpires),
		})
	}
	return out, nil
}

// RenewSubdomain 续期指定免费子域名。
// 未到官方续期窗口时 DNSHE 返回 422 + error_code=renewal_not_yet_available，
// 调用方可通过错误文本中的该错误码识别并跳过。
func (d *DNSHE) RenewSubdomain(ctx context.Context, cred Credentials, subdomainID int64) (*RenewResult, error) {
	payload := map[string]any{"subdomain_id": subdomainID}
	var resp dnsheRenewResponse
	if err := d.call(ctx, cred, http.MethodPost, "subdomains", "renew", nil, payload, &resp); err != nil {
		return nil, err
	}
	result := RenewResult{
		FullDomain:        resp.FullDomain,
		PreviousExpiresAt: resp.PreviousExpiresAt,
		NewExpiresAt:      resp.NewExpiresAt,
		NeverExpires:      bool(resp.NeverExpires),
		Status:            resp.Status,
		RemainingDays:     resp.RemainingDays,
	}
	// 兼容字段嵌套在 data 下的响应
	if resp.Data != nil {
		if result.FullDomain == "" {
			result.FullDomain = resp.Data.FullDomain
		}
		if result.PreviousExpiresAt == "" {
			result.PreviousExpiresAt = resp.Data.PreviousExpiresAt
		}
		if result.NewExpiresAt == "" {
			result.NewExpiresAt = resp.Data.NewExpiresAt
		}
		if result.Status == "" {
			result.Status = resp.Data.Status
		}
		if resp.Data.NeverExpires {
			result.NeverExpires = true
		}
		if result.RemainingDays == 0 {
			result.RemainingDays = resp.Data.RemainingDays
		}
	}
	// 续期后使列表缓存失效，保证下次拉取拿到新的到期时间
	d.cacheMu.Lock()
	delete(d.cache, cred.Token)
	d.cacheMu.Unlock()
	return &result, nil
}

type dnsheRenewFields struct {
	FullDomain        string        `json:"full_domain"`
	Domain            string        `json:"domain"`
	PreviousExpiresAt string        `json:"previous_expires_at"`
	NewExpiresAt      string        `json:"new_expires_at"`
	NeverExpires      dnsheFlexBool `json:"never_expires"`
	Status            string        `json:"status"`
	RemainingDays     int           `json:"remaining_days"`
}

type dnsheRenewResponse struct {
	dnsheBase
	dnsheRenewFields
	Data *dnsheRenewFields `json:"data"`
}
