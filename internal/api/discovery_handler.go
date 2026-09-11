package api

import (
	"net/http"

	"github.com/fonu/fonu/internal/discovery"
)

type DiscoveryHandler struct {
	svc *discovery.Service
}

func NewDiscoveryHandler(svc *discovery.Service) *DiscoveryHandler {
	return &DiscoveryHandler{svc: svc}
}

func (h *DiscoveryHandler) Scan(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	writeJSON(w, http.StatusOK, h.svc.Scan(r.Context(), host))
}
