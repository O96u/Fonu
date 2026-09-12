package api

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/logstore"
)

type LogsHandler struct {
	cfg config.Config
}

func NewLogsHandler(cfg config.Config) *LogsHandler {
	return &LogsHandler{cfg: cfg}
}

func (h *LogsHandler) Access(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 100)
	keyword := r.URL.Query().Get("keyword")
	status := queryInt(r, "status", 0)
	entries, err := logstore.ReadAccess(filepath.Join(h.cfg.LogsDir(), "access.log"), limit, keyword, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取访问日志失败")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *LogsHandler) Error(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 100)
	keyword := r.URL.Query().Get("keyword")
	lines, err := logstore.ReadError(filepath.Join(h.cfg.LogsDir(), "error.log"), limit, keyword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取错误日志失败")
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
		writeError(w, http.StatusInternalServerError, "读取系统日志失败")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *LogsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("type")
	if kind == "" {
		kind = "system"
	}
	path := filepath.Join(h.cfg.LogsDir(), "app.log")
	switch kind {
	case "access":
		path = filepath.Join(h.cfg.LogsDir(), "access.log")
	case "error":
		path = filepath.Join(h.cfg.LogsDir(), "error.log")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "SSE 不可用")
		return
	}
	tail := queryInt(r, "tail", 100)
	_ = logstore.StreamFile(r.Context(), path, w, func() error {
		flusher.Flush()
		return nil
	}, tail)
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
