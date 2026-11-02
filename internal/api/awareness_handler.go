package api

import (
	"net/http"

	"github.com/fonu/fonu/internal/awareness"
	"github.com/fonu/fonu/internal/iplocate"
)

type AwarenessHandler struct {
	analyzer *awareness.Analyzer
	geo      *iplocate.Service
}

func NewAwarenessHandler(analyzer *awareness.Analyzer, geo *iplocate.Service) *AwarenessHandler {
	return &AwarenessHandler{analyzer: analyzer, geo: geo}
}

func (h *AwarenessHandler) Live(w http.ResponseWriter, r *http.Request) {
	if h.analyzer == nil {
		writeJSON(w, http.StatusOK, awareness.LiveSnapshot{})
		return
	}
	rangeKey := r.URL.Query().Get("range")
	snap := h.analyzer.LiveForRange(rangeKey)
	awareness.EnrichEventsGeo(snap.RecentEvents, h.geo)
	writeJSON(w, http.StatusOK, snap)
}
