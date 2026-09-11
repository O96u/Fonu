package api

import (
	"encoding/json"
	"net/http"

	"github.com/fonu/fonu/internal/acme"
)

type CertHandler struct {
	svc *acme.Service
}

func NewCertHandler(svc *acme.Service) *CertHandler {
	return &CertHandler{svc: svc}
}

func (h *CertHandler) List(w http.ResponseWriter, r *http.Request) {
	records, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取证书失败")
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (h *CertHandler) Apply(w http.ResponseWriter, r *http.Request) {
	records, err := h.svc.Apply(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (h *CertHandler) Renew(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	record, err := h.svc.Renew(r.Context(), req.Domain)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}
