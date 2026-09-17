package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/service"
	"github.com/fonu/fonu/internal/traffic"
)

type ProxyHandler struct {
	cfg     config.Config
	svc     *service.ProxyService
	logs    *LogsHandler
	traffic *traffic.Collector
}

func NewProxyHandler(cfg config.Config, svc *service.ProxyService, logs *LogsHandler, collector *traffic.Collector) *ProxyHandler {
	return &ProxyHandler{cfg: cfg, svc: svc, logs: logs, traffic: collector}
}

type proxyRequest struct {
	Domain       string           `json:"domain"`
	Upstream     string           `json:"upstream"`
	ListenPort   *int             `json:"listen_port"`
	ListenIPv4   *bool            `json:"listen_ipv4"`
	ListenIPv6   *bool            `json:"listen_ipv6"`
	Hosts        []string         `json:"hosts"`
	HTTPSEnabled *bool            `json:"https_enabled"`
	HTTPRedirect *bool            `json:"http_redirect"`
	Enabled      *bool            `json:"enabled"`
	Name         *string          `json:"name"`
	Remark       *string          `json:"remark"` // deprecated alias for name
	Security     *securityRequest `json:"security,omitempty"`
}

type proxyReorderRequest struct {
	IDs []int64 `json:"ids"`
}

func (h *ProxyHandler) List(w http.ResponseWriter, r *http.Request) {
	rules, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取反向代理规则失败")
		return
	}
	writeJSON(w, http.StatusOK, proxy.RulesForAPI(rules))
}

func (h *ProxyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req proxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}

	hosts := req.hosts()
	if len(hosts) == 0 {
		writeError(w, http.StatusBadRequest, "至少需要一个前端域名")
		return
	}

	listenPort := intDefault(req.ListenPort, h.cfg.DefaultListenPort(boolDefault(req.HTTPSEnabled, true)))
	name := ""
	if resolved := req.resolveName(); resolved != nil {
		name = *resolved
	}
	in := proxy.CreateInput{
		Upstream:     req.Upstream,
		ListenPort:   listenPort,
		ListenIPv4:   boolDefault(req.ListenIPv4, true),
		ListenIPv6:   boolDefault(req.ListenIPv6, false),
		Hosts:        hosts,
		HTTPSEnabled: boolDefault(req.HTTPSEnabled, true),
		HTTPRedirect: boolDefault(req.HTTPRedirect, true),
		Enabled:      boolDefault(req.Enabled, true),
		Name:         name,
		Security:     req.securityInput(),
	}

	rule, err := h.svc.Create(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, proxy.RuleForAPI(rule))
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
	if req.Upstream != "" {
		in.Upstream = &req.Upstream
	}
	if req.ListenPort != nil {
		in.ListenPort = req.ListenPort
	}
	if req.ListenIPv4 != nil {
		in.ListenIPv4 = req.ListenIPv4
	}
	if req.ListenIPv6 != nil {
		in.ListenIPv6 = req.ListenIPv6
	}
	if hosts := req.hosts(); len(hosts) > 0 {
		in.Hosts = &hosts
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
	if resolved := req.resolveName(); resolved != nil {
		in.Name = resolved
	}
	if req.Security != nil {
		sec := req.securityInput()
		in.Security = &sec
	}

	rule, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, proxy.RuleForAPI(rule))
}

func (h *ProxyHandler) Traffic(w http.ResponseWriter, r *http.Request) {
	if h.traffic == nil {
		writeJSON(w, http.StatusOK, []traffic.RuleTraffic{})
		return
	}
	rules, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取反向代理规则失败")
		return
	}
	writeJSON(w, http.StatusOK, h.traffic.SnapshotForRules(rules))
}

func (h *ProxyHandler) Clients(w http.ResponseWriter, r *http.Request) {
	if h.traffic == nil {
		writeJSON(w, http.StatusOK, []traffic.ClientConn{})
		return
	}
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的规则 ID")
		return
	}
	rule, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "规则不存在")
		return
	}
	writeJSON(w, http.StatusOK, h.traffic.ClientsForRule(rule))
}

func (h *ProxyHandler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的规则 ID")
		return
	}
	rule, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "规则不存在")
		return
	}
	hosts := rule.Hostnames()
	if len(hosts) == 0 {
		writeError(w, http.StatusBadRequest, "该规则没有前端域名")
		return
	}
	h.logs.StreamAccessForHosts(w, r, hosts)
}

func (h *ProxyHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req proxyReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	if err := h.svc.Reorder(r.Context(), req.IDs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (req proxyRequest) resolveName() *string {
	if req.Name != nil {
		return req.Name
	}
	return req.Remark
}

func (req proxyRequest) hosts() []string {
	if len(req.Hosts) > 0 {
		return req.Hosts
	}
	if req.Domain != "" {
		return []string{req.Domain}
	}
	return nil
}

func intDefault(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
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
