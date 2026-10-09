package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fonu/fonu/internal/ddns"
	"github.com/fonu/fonu/internal/dnsherenew"
)

type DNSHEHandler struct {
	svc     *dnsherenew.Service
	ddnsSvc *ddns.Service
}

func NewDNSHEHandler(svc *dnsherenew.Service, ddnsSvc *ddns.Service) *DNSHEHandler {
	return &DNSHEHandler{svc: svc, ddnsSvc: ddnsSvc}
}

// ---- 账户 ----

func (h *DNSHEHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.svc.ListAccounts(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}

type dnsheAccountRequest struct {
	Name       string `json:"name"`
	APIKey     string `json:"api_key"`
	APISecret  string `json:"api_secret"`
	AutoRenew  bool   `json:"auto_renew"`
}

func (h *DNSHEHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req dnsheAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	account, err := h.svc.CreateAccount(r.Context(), dnsherenew.AccountInput{
		Name: req.Name, APIKey: req.APIKey, APISecret: req.APISecret, AutoRenew: req.AutoRenew,
	})
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

func (h *DNSHEHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	var req dnsheAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	account, err := h.svc.UpdateAccount(r.Context(), r.PathValue("id"), dnsherenew.AccountInput{
		Name: req.Name, APIKey: req.APIKey, APISecret: req.APISecret, AutoRenew: req.AutoRenew,
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dnsherenew.ErrAccountNotFound) {
			status = http.StatusNotFound
		}
		writeError(r, w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, account)
}

func (h *DNSHEHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteAccount(r.Context(), r.PathValue("id")); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dnsherenew.ErrAccountNotFound) {
			status = http.StatusNotFound
		}
		writeError(r, w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "已删除"})
}

func (h *DNSHEHandler) ReorderAccounts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if len(req.IDs) == 0 {
		writeError(r, w, http.StatusBadRequest, "排序数据不能为空")
		return
	}
	if err := h.svc.ReorderAccounts(r.Context(), req.IDs); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "已保存排序"})
}

// ---- 域名 ----

// RevealCredentials 返回账户解密后的 API 凭据（DNSHE 官网只在创建时显示一次）。
func (h *DNSHEHandler) RevealCredentials(w http.ResponseWriter, r *http.Request) {
	cred, err := h.svc.RevealCredentials(r.Context(), r.PathValue("id"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dnsherenew.ErrAccountNotFound) {
			status = http.StatusNotFound
		}
		writeError(r, w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"api_key":    cred.Token,
		"api_secret": cred.Secret,
	})
}

// PushToDDNS 用账户凭据把其下域名按主域名分组创建 DDNS 任务，已有主域名的任务跳过。
func (h *DNSHEHandler) PushToDDNS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountID := r.PathValue("id")

	cred, err := h.svc.RevealCredentials(ctx, accountID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, dnsherenew.ErrAccountNotFound) {
			status = http.StatusNotFound
		}
		writeError(r, w, status, err.Error())
		return
	}

	domains, err := h.svc.ListDomains(ctx, accountID)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	if len(domains) == 0 {
		writeError(r, w, http.StatusBadRequest, "该账户下没有域名，无可推送")
		return
	}

	// 按主域名分组：full_domain 末两段为主域名，其余为记录名
	grouped := map[string][]string{}
	var roots []string
	for _, d := range domains {
		full := strings.ToLower(strings.TrimSpace(d.FullDomain))
		parts := strings.Split(full, ".")
		if len(parts) < 3 {
			continue
		}
		root := strings.Join(parts[len(parts)-2:], ".")
		name := strings.Join(parts[:len(parts)-2], ".")
		if _, ok := grouped[root]; !ok {
			roots = append(roots, root)
		}
		dup := false
		for _, n := range grouped[root] {
			if n == name {
				dup = true
				break
			}
		}
		if !dup {
			grouped[root] = append(grouped[root], name)
		}
	}
	if len(roots) == 0 {
		writeError(r, w, http.StatusBadRequest, "未解析出可推送的域名")
		return
	}

	existing, err := h.ddnsSvc.ListLite(ctx)
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取 DDNS 配置失败")
		return
	}
	existingRoots := map[string]bool{}
	for _, c := range existing {
		existingRoots[strings.ToLower(c.RootDomain)] = true
	}

	var created, skipped []string
	for _, root := range roots {
		if existingRoots[root] {
			skipped = append(skipped, root)
			continue
		}
		_, err := h.ddnsSvc.Create(ctx, ddns.SaveInput{
			Provider:    "dnshe",
			RootDomain:  root,
			RecordNames: grouped[root],
			IPv4Enabled: true,
			Enabled:     true,
			APIToken:    cred.Token,
			APISecret:   cred.Secret,
		})
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s（%s）", root, err.Error()))
			continue
		}
		created = append(created, root)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"created": created,
		"skipped": skipped,
		"message": fmt.Sprintf("已创建 %d 个 DDNS 任务，跳过 %d 个", len(created), len(skipped)),
	})
}

func (h *DNSHEHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := h.svc.ListDomains(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domains)
}

func (h *DNSHEHandler) RenewDomain(w http.ResponseWriter, r *http.Request) {
	subID, err := strconv.ParseInt(r.PathValue("subId"), 10, 64)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, "无效的域名 ID")
		return
	}
	result, err := h.svc.RenewDomain(r.Context(), r.PathValue("id"), subID)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *DNSHEHandler) DeleteDomain(w http.ResponseWriter, r *http.Request) {
	subID, err := strconv.ParseInt(r.PathValue("subId"), 10, 64)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, "无效的域名 ID")
		return
	}
	if err := h.svc.DeleteDomain(r.Context(), r.PathValue("id"), subID); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "域名已删除"})
}

// SetDomainAutoRenew 设置单个域名的自动续期开关。
func (h *DNSHEHandler) SetDomainAutoRenew(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullDomain string `json:"full_domain"`
		Enabled    bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if err := h.svc.SetDomainAutoRenew(r.Context(), req.FullDomain, req.Enabled); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "已保存"})
}

// ReorderDomains 保存域名显示顺序。
func (h *DNSHEHandler) ReorderDomains(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if len(req.IDs) == 0 {
		writeError(r, w, http.StatusBadRequest, "排序数据不能为空")
		return
	}
	if err := h.svc.ReorderDomains(r.Context(), r.PathValue("id"), req.IDs); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "已保存排序"})
}

// RegisterDomain 注册新域名。
func (h *DNSHEHandler) RegisterDomain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Subdomain string `json:"subdomain"`
		Domain    string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	result, err := h.svc.RegisterDomain(r.Context(), r.PathValue("id"), req.Subdomain, req.Domain)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// ---- 转赠 / 接收 ----

func (h *DNSHEHandler) ListGifts(w http.ResponseWriter, r *http.Request) {
	gifts, err := h.svc.ListGifts(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gifts)
}

func (h *DNSHEHandler) InitiateGift(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SubdomainID int64 `json:"subdomain_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	gift, err := h.svc.InitiateGift(r.Context(), r.PathValue("id"), req.SubdomainID)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, gift)
}

func (h *DNSHEHandler) AcceptGift(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	gift, err := h.svc.AcceptGift(r.Context(), r.PathValue("id"), req.Code)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gift)
}

func (h *DNSHEHandler) CancelGift(w http.ResponseWriter, r *http.Request) {
	giftID, err := strconv.ParseInt(r.PathValue("giftId"), 10, 64)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, "无效的转赠 ID")
		return
	}
	if err := h.svc.CancelGift(r.Context(), r.PathValue("id"), giftID); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "转赠已取消"})
}
