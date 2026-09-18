package api

import (
	"net/http"

	"github.com/fonu/fonu/internal/chinacidr"
)

type ChinaCIDRHandler struct {
	svc *chinacidr.Service
}

func NewChinaCIDRHandler(svc *chinacidr.Service) *ChinaCIDRHandler {
	return &ChinaCIDRHandler{svc: svc}
}

func (h *ChinaCIDRHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Status(r.Context()))
}

func (h *ChinaCIDRHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.svc.Update(ctx); err != nil {
		h.svc.RecordLastError(ctx, err)
		writeError(r, w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.svc.Status(ctx))
}
