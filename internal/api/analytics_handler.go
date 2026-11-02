package api

import (
	"net/http"

	"github.com/fonu/fonu/internal/analytics"
)

type AnalyticsHandler struct {
	svc *analytics.Service
}

func NewAnalyticsHandler(svc *analytics.Service) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

func (h *AnalyticsHandler) Overview(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeJSON(w, http.StatusOK, analytics.Overview{})
		return
	}
	rangeKey := r.URL.Query().Get("range")
	out, err := h.svc.Overview(r.Context(), rangeKey)
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取访问统计失败", err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
