package api

import (
	"encoding/json"
	"net/http"
	"strconv"

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

func (h *DDNSHandler) toInput(req ddnsRequest) ddns.SaveInput {
	return ddns.SaveInput{
		Provider:    req.Provider,
		RootDomain:  req.RootDomain,
		RecordName:  req.RecordName,
		IPv4Enabled: boolDefault(req.IPv4Enabled, true),
		IPv6Enabled: boolDefault(req.IPv6Enabled, false),
		Enabled:     boolDefault(req.Enabled, true),
		APIToken:    req.APIToken,
		APITokenID:  req.APITokenID,
		APISecret:   req.APISecret,
	}
}

func (h *DDNSHandler) List(w http.ResponseWriter, r *http.Request) {
	configs, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取 DDNS 配置失败")
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

func (h *DDNSHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ddnsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	cfg, err := h.svc.Create(r.Context(), h.toInput(req))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *DDNSHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的配置 ID")
		return
	}
	var req ddnsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	cfg, err := h.svc.Update(r.Context(), id, h.toInput(req))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *DDNSHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的配置 ID")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DDNSHandler) Test(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConfigID   int64  `json:"config_id"`
		Provider   string `json:"provider"`
		APIToken   string `json:"api_token"`
		APITokenID string `json:"api_token_id"`
		APISecret  string `json:"api_secret"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if err := h.svc.Test(r.Context(), ddns.TestInput{
		ConfigID:   req.ConfigID,
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

func (h *DDNSHandler) UpdateOne(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的配置 ID")
		return
	}
	cfg, err := h.svc.UpdateNow(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *DDNSHandler) UpdateAll(w http.ResponseWriter, r *http.Request) {
	configs, err := h.svc.UpdateAll(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, configs)
}
