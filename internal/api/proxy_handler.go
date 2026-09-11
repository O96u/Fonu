package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/service"
)

type ProxyHandler struct {
	svc *service.ProxyService
}

func NewProxyHandler(svc *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{svc: svc}
}

type proxyRequest struct {
	Domain       string `json:"domain"`
	Upstream     string `json:"upstream"`
	HTTPSEnabled *bool  `json:"https_enabled"`
	HTTPRedirect *bool  `json:"http_redirect"`
	Enabled      *bool  `json:"enabled"`
}

func (h *ProxyHandler) List(w http.ResponseWriter, r *http.Request) {
	rules, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取反向代理规则失败")
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *ProxyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req proxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}

	in := proxy.CreateInput{
		Domain:       req.Domain,
		Upstream:     req.Upstream,
		HTTPSEnabled: boolDefault(req.HTTPSEnabled, true),
		HTTPRedirect: boolDefault(req.HTTPRedirect, true),
		Enabled:      boolDefault(req.Enabled, true),
	}

	rule, err := h.svc.Create(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *ProxyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的规则 ID")
		return
	}

	var req proxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}

	in := proxy.UpdateInput{}
	if req.Domain != "" {
		in.Domain = &req.Domain
	}
	if req.Upstream != "" {
		in.Upstream = &req.Upstream
	}
	if req.HTTPSEnabled != nil {
		in.HTTPSEnabled = req.HTTPSEnabled
	}
	if req.HTTPRedirect != nil {
		in.HTTPRedirect = req.HTTPRedirect
	}
	if req.Enabled != nil {
		in.Enabled = req.Enabled
	}

	rule, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *ProxyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的规则 ID")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func boolDefault(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}
