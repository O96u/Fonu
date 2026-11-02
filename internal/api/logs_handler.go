package api

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fonu/fonu/internal/awareness"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/iplocate"
	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/service"
)

type LogsHandler struct {
	cfg      config.Config
	proxySvc *service.ProxyService
	geo      *iplocate.Service
}

func NewLogsHandler(cfg config.Config, proxySvc *service.ProxyService, geo *iplocate.Service) *LogsHandler {
	return &LogsHandler{cfg: cfg, proxySvc: proxySvc, geo: geo}
}

type accessLogRow struct {
	logstore.AccessEntry
	Geo string `json:"geo,omitempty"`
}

func (h *LogsHandler) enrichAccessRows(entries []logstore.AccessEntry) []accessLogRow {
	out := make([]accessLogRow, len(entries))
	cache := map[string]string{}
	for i := range entries {
		out[i].AccessEntry = entries[i]
		if h.geo != nil {
			out[i].Geo = h.geo.LookupCached(entries[i].ClientIP, cache)
		}
	}
	return out
}

func (h *LogsHandler) enrichFilteredRows(entries []awareness.EnrichedAccessLog) {
	if h.geo == nil {
		return
	}
	cache := map[string]string{}
	for i := range entries {
		entries[i].Geo = h.geo.LookupCached(entries[i].ClientIP, cache)
	}
}

func (h *LogsHandler) Access(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 100)
	keyword := r.URL.Query().Get("keyword")
	status := queryInt(r, "status", 0)
	clientIP := strings.TrimSpace(r.URL.Query().Get("ip"))
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))

	path := filepath.Join(h.cfg.LogsDir(), "access.log")
	if kind != "" || clientIP != "" {
		var bindings proxy.BindingIndex
		if h.proxySvc != nil {
			if rules, err := h.proxySvc.List(r.Context()); err == nil {
				bindings = proxy.BuildBindingIndex(rules)
			}
		}
		entries, err := awareness.FilterAccessLog(path, awareness.AccessLogFilter{
			Limit:    limit,
			Keyword:  keyword,
			Status:   status,
			ClientIP: clientIP,
			Kind:     kind,
		}, bindings)
		if err != nil {
			writeError(r, w, http.StatusInternalServerError, "读取访问日志失败")
			return
		}
		h.enrichFilteredRows(entries)
		writeJSON(w, http.StatusOK, entries)
		return
	}

	entries, err := logstore.ReadAccess(path, limit, keyword, status)
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取访问日志失败")
		return
	}
	writeJSON(w, http.StatusOK, h.enrichAccessRows(entries))
}

func (h *LogsHandler) Error(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 100)
	keyword := r.URL.Query().Get("keyword")
	lines, err := logstore.ReadError(filepath.Join(h.cfg.LogsDir(), "error.log"), limit, keyword)
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取错误日志失败")
		return
	}
	writeJSON(w, http.StatusOK, lines)
}

func (h *LogsHandler) System(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 100)
	level := r.URL.Query().Get("level")
	keyword := r.URL.Query().Get("keyword")
	entries, err := logstore.ReadSystem(filepath.Join(h.cfg.LogsDir(), "app.log"), limit, level, keyword)
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取系统日志失败")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *LogsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("type")
	if kind == "" {
		kind = "error"
	}
	var path string
	switch kind {
	case "access":
		path = filepath.Join(h.cfg.LogsDir(), "access.log")
	case "error":
		path = filepath.Join(h.cfg.LogsDir(), "error.log")
	default:
		writeError(r, w, http.StatusBadRequest, "实时日志仅支持 Nginx 访问/错误日志")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(r, w, http.StatusInternalServerError, "SSE 不可用")
		return
	}
	tail := queryInt(r, "tail", 100)
	var filter logstore.LineFilter
	if kind == "access" {
		filter = logstore.AccessHostFilter(r.URL.Query()["host"])
	}
	_ = logstore.StreamFile(r.Context(), path, w, func() error {
		flusher.Flush()
		return nil
	}, tail, filter)
}

func (h *LogsHandler) StreamAccessForHosts(w http.ResponseWriter, r *http.Request, hosts []string) {
	endpoints := make([]logstore.AccessEndpoint, 0, len(hosts))
	for _, host := range hosts {
		endpoints = append(endpoints, logstore.AccessEndpoint{Host: host})
	}
	h.StreamAccessForEndpoints(w, r, endpoints)
}

func (h *LogsHandler) StreamAccessForEndpoints(w http.ResponseWriter, r *http.Request, endpoints []logstore.AccessEndpoint) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(r, w, http.StatusInternalServerError, "SSE 不可用")
		return
	}
	path := filepath.Join(h.cfg.LogsDir(), "access.log")
	tail := queryInt(r, "tail", 100)
	filter := logstore.AccessEndpointFilter(endpoints)
	_ = logstore.StreamFile(r.Context(), path, w, func() error {
		flusher.Flush()
		return nil
	}, tail, filter)
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
