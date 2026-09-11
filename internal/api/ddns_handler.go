package api

import (
	"encoding/json"
	"net/http"

	"github.com/fonu/fonu/internal/ddns"
)

type DDNSHandler struct {
	svc *ddns.Service
}

func NewDDNSHandler(svc *ddns.Service) *DDNSHandler {
	return &DDNSHandler{svc: svc}
}

type ddnsRequest struct {
	Provider    string `json:"provider"`
	RootDomain  string `json:"root_domain"`
	RecordName  string `json:"record_name"`
	IPv4Enabled *bool  `json:"ipv4_enabled"`
	IPv6Enabled *bool  `json:"ipv6_enabled"`
	Enabled     *bool  `json:"enabled"`
	APIToken    string `json:"api_token"`
	APITokenID  string `json:"api_token_id"`
	APISecret   string `json:"api_secret"`
}

func (h *DDNSHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.svc.Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"configured": false,
		})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *DDNSHandler) Put(w http.ResponseWriter, r *http.Request) {
	var req ddnsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	cfg, err := h.svc.Save(r.Context(), ddns.SaveInput{
		Provider:    req.Provider,
		RootDomain:  req.RootDomain,
		RecordName:  req.RecordName,
		IPv4Enabled: boolDefault(req.IPv4Enabled, true),
		IPv6Enabled: boolDefault(req.IPv6Enabled, false),
		Enabled:     boolDefault(req.Enabled, true),
		APIToken:    req.APIToken,
		APITokenID:  req.APITokenID,
		APISecret:   req.APISecret,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *DDNSHandler) Test(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider   string `json:"provider"`
		APIToken   string `json:"api_token"`
		APITokenID string `json:"api_token_id"`
		APISecret  string `json:"api_secret"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := h.svc.Test(r.Context(), ddns.TestInput{
		Provider:   req.Provider,
		APIToken:   req.APIToken,
		APITokenID: req.APITokenID,
		APISecret:  req.APISecret,
	}); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "连接成功"})
}

func (h *DDNSHandler) Update(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.UpdateNow(r.Context()); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := h.svc.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取 DDNS 状态失败")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}
