package api

import (
	"net/http"

	"github.com/fonu/fonu/internal/analytics"
	"github.com/fonu/fonu/internal/awareness"
	"github.com/fonu/fonu/internal/iplocate"
)

type AwarenessHandler struct {
	analytics *analytics.Service
	geo       *iplocate.Service
}

func NewAwarenessHandler(analyticsSvc *analytics.Service, geo *iplocate.Service) *AwarenessHandler {
	return &AwarenessHandler{analytics: analyticsSvc, geo: geo}
}

func (h *AwarenessHandler) Live(w http.ResponseWriter, r *http.Request) {
	if h.analytics == nil {
		writeJSON(w, http.StatusOK, awareness.LiveSnapshot{})
		return
	}
	rangeKey := r.URL.Query().Get("range")
	snap, err := h.analytics.LiveSnapshot(r.Context(), rangeKey)
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取动态感知失败", err)
		return
	}
	awareness.EnrichEventsGeo(snap.RecentEvents, h.geo)
	writeJSON(w, http.StatusOK, snap)
}
