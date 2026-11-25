package ddns

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Gift 是 DNSHE 域名转赠记录。
// 流程：转赠方 InitiateGift 生成码（3 天内有效，域名需注册满 30 天），
// 接收方凭码 AcceptGift 完成域名过户；转赠方可在被接收前 CancelGift。
type Gift struct {
	ID         int64  `json:"id"`
	Code       string `json:"code"`
	FullDomain string `json:"full_domain"`
	Status     string `json:"status"` // pending / accepted / cancelled / expired
	FromUserID int64  `json:"from_userid"`
	ToUserID   int64  `json:"to_userid"`
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at"`
}

// DeleteSubdomain 删除免费子域名（同时删除其 DNS 记录），不可恢复。
func (d *DNSHE) DeleteSubdomain(ctx context.Context, cred Credentials, subdomainID int64) error {
	payload := map[string]any{"subdomain_id": subdomainID}
	var resp dnsheMessageResponse
	if err := d.call(ctx, cred, http.MethodPost, "subdomains", "delete", nil, payload, &resp); err != nil {
		// DNSHE 对不存在的 ID 返回 404 HTML，转成语义化错误
		if strings.Contains(err.Error(), "HTTP 404") {
			return fmt.Errorf("域名不存在或已被删除")
		}
		return err
	}
	d.cacheMu.Lock()
	delete(d.cache, cred.Token)
	d.cacheMu.Unlock()
	return nil
}

// InitiateGift 生成转赠码。域名注册未满 30 天时返回 422。
func (d *DNSHE) InitiateGift(ctx context.Context, cred Credentials, subdomainID int64) (*Gift, error) {
	payload := map[string]any{"subdomain_id": subdomainID}
	var resp dnsheGiftResponse
	if err := d.call(ctx, cred, http.MethodPost, "gifts", "initiate", nil, payload, &resp); err != nil {
		return nil, err
	}
	gift := resp.result()
	return &gift, nil
}

// AcceptGift 凭转赠码接收域名。
func (d *DNSHE) AcceptGift(ctx context.Context, cred Credentials, code string) (*Gift, error) {
	payload := map[string]any{"code": strings.TrimSpace(code)}
	var resp dnsheGiftResponse
	if err := d.call(ctx, cred, http.MethodPost, "gifts", "accept", nil, payload, &resp); err != nil {
		return nil, err
	}
	d.cacheMu.Lock()
	delete(d.cache, cred.Token)
	d.cacheMu.Unlock()
	gift := resp.result()
	return &gift, nil
}

// CancelGift 取消尚未被接收的转赠。
func (d *DNSHE) CancelGift(ctx context.Context, cred Credentials, giftID int64) error {
	payload := map[string]any{"gift_id": giftID}
	var resp dnsheMessageResponse
	return d.call(ctx, cred, http.MethodPost, "gifts", "cancel", nil, payload, &resp)
}

// ListGifts 列出当前账户的转赠记录（含已接收/已取消）。
func (d *DNSHE) ListGifts(ctx context.Context, cred Credentials) ([]Gift, error) {
	var resp dnsheGiftsResponse
	if err := d.call(ctx, cred, http.MethodGet, "gifts", "list", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Gifts, nil
}

// DNSHE mutation responses may wrap the gift in data or gift.
type dnsheGiftResponse struct {
	Gift
	Data *struct {
		Gift
		Nested *Gift `json:"gift"`
	} `json:"data"`
	Nested *Gift `json:"gift"`
}

func (r dnsheGiftResponse) result() Gift {
	result := r.Gift
	for _, g := range []*Gift{r.Nested, func() *Gift { if r.Data != nil { return &r.Data.Gift }; return nil }(), func() *Gift { if r.Data != nil { return r.Data.Nested }; return nil }()} {
		if g == nil { continue }
		if result.ID == 0 { result.ID = g.ID }
		if result.Code == "" { result.Code = g.Code }
		if result.FullDomain == "" { result.FullDomain = g.FullDomain }
		if result.Status == "" { result.Status = g.Status }
		if result.ExpiresAt == "" { result.ExpiresAt = g.ExpiresAt }
		if result.CreatedAt == "" { result.CreatedAt = g.CreatedAt }
		if result.FromUserID == 0 { result.FromUserID = g.FromUserID }
		if result.ToUserID == 0 { result.ToUserID = g.ToUserID }
	}
	return result
}

type dnsheGiftsResponse struct {
	dnsheBase
	Gifts []Gift `json:"gifts"`
}
