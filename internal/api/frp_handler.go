package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fonu/fonu/internal/frp"
	"github.com/fonu/fonu/internal/service"
)

type FRPHandler struct {
	mgr   *frp.Manager
	proxy *service.ProxyService
}

func NewFRPHandler(mgr *frp.Manager, proxySvc *service.ProxyService) *FRPHandler {
	return &FRPHandler{mgr: mgr, proxy: proxySvc}
}

type frpResponse struct {
	frp.Config
	Status         frp.Status `json:"status"`
	FRPSConfig     string     `json:"frps_config"`
	NginxHTTPPort  int        `json:"nginx_http_port"`
	NginxHTTPSPort int        `json:"nginx_https_port"`
}

type frpSaveRequest struct {
	Enabled       bool           `json:"enabled"`
	ServerAddr    string         `json:"server_addr"`
	ServerPort    int            `json:"server_port"`
	AuthToken     string         `json:"auth_token"`
	TLSEnabled    bool           `json:"tls_enabled"`
	CustomDomains []string       `json:"custom_domains"`
	TCPProxies    []frp.TCPProxy `json:"tcp_proxies"`
}

func (h *FRPHandler) buildResponse(ctx context.Context, cfg frp.Config) frpResponse {
	frpsConfig, err := h.mgr.FRPSConfig(ctx)
	if err != nil {
		frpsConfig = ""
	}
	return frpResponse{
		Config:         cfg,
		Status:         h.mgr.Status(ctx),
		FRPSConfig:     frpsConfig,
		NginxHTTPPort:  h.mgr.NginxHTTPPort(),
		NginxHTTPSPort: h.mgr.NginxHTTPSPort(),
	}
}

func (h *FRPHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.mgr.Load(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取 FRP 配置失败")
		return
	}
	writeJSON(w, http.StatusOK, h.buildResponse(r.Context(), cfg))
}

func (h *FRPHandler) Put(w http.ResponseWriter, r *http.Request) {
	var req frpSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "请求格式无效")
		return
	}
	result, err := h.mgr.Apply(r.Context(), frp.SaveInput{
		Enabled:       req.Enabled,
		ServerAddr:    req.ServerAddr,
		ServerPort:    req.ServerPort,
		AuthToken:     req.AuthToken,
		TLSEnabled:    req.TLSEnabled,
		CustomDomains: req.CustomDomains,
		TCPProxies:    req.TCPProxies,
	})
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := h.mgr.Load(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取 FRP 配置失败")
		return
	}
	resp := h.buildResponse(r.Context(), cfg)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":          result.Message,
		"config":           resp.Config,
		"status":           resp.Status,
		"frps_config":      resp.FRPSConfig,
		"nginx_http_port":  resp.NginxHTTPPort,
		"nginx_https_port": resp.NginxHTTPSPort,
	})
}

func (h *FRPHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.mgr.Status(r.Context()))
}

func (h *FRPHandler) SyncDomains(w http.ResponseWriter, r *http.Request) {
	rules, err := h.proxy.List(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取反代规则失败")
		return
	}
	domains := frp.DomainsFromProxyRules(rules)
	if len(domains) == 0 {
		writeError(r, w, http.StatusBadRequest, "没有可同步的启用反代域名")
		return
	}
	result, err := h.mgr.SyncDomains(r.Context(), domains)
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := h.mgr.Load(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取 FRP 配置失败")
		return
	}
	resp := h.buildResponse(r.Context(), cfg)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":          result.Message,
		"domains":          cfg.CustomDomains,
		"config":           resp.Config,
		"status":           resp.Status,
		"frps_config":      resp.FRPSConfig,
		"nginx_http_port":  resp.NginxHTTPPort,
		"nginx_https_port": resp.NginxHTTPSPort,
	})
}

func (h *FRPHandler) Logs(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 200)
	lines, err := h.mgr.Logs(r.Context(), limit)
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取 FRP 日志失败")
		return
	}
	writeJSON(w, http.StatusOK, lines)
}
