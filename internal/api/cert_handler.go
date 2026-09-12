package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

func (h *CertHandler) Options(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]string{
		{"value": acme.CALetsEncrypt, "label": acme.CALabel(acme.CALetsEncrypt)},
		{"value": acme.CALetsEncryptStaging, "label": acme.CALabel(acme.CALetsEncryptStaging)},
	})
}

func (h *CertHandler) Apply(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain  string   `json:"domain"`
		DNSZone string   `json:"dns_zone"`
		Domains []string `json:"domains"`
		CA      string   `json:"ca"`
		Email   string   `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	domains := req.Domains
	if len(domains) == 0 && strings.TrimSpace(req.Domain) != "" {
		domains = []string{req.Domain}
	}
	dnsZone := strings.TrimSpace(req.DNSZone)
	if dnsZone == "" && len(domains) > 0 {
		dnsZone = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domains[0])), "*.")
	}
	jobID, err := h.svc.StartApply(r.Context(), dnsZone, domains, req.CA, req.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

func (h *CertHandler) ApplyJobStream(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("id"))
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "无效的任务 ID")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "SSE 不可用")
		return
	}
	if err := h.svc.StreamJob(r.Context(), w, jobID, func() error {
		flusher.Flush()
		return nil
	}); err != nil {
		return
	}
}

func (h *CertHandler) Import(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Certificate string `json:"certificate"`
		PrivateKey  string `json:"private_key"`
		CertPath    string `json:"cert_path"`
		KeyPath     string `json:"key_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式无效")
		return
	}
	record, err := h.svc.Import(r.Context(), req.Certificate, req.PrivateKey, req.CertPath, req.KeyPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (h *CertHandler) Download(w http.ResponseWriter, r *http.Request) {
	domain := strings.TrimSpace(r.PathValue("domain"))
	if domain == "" {
		writeError(w, http.StatusBadRequest, "域名不能为空")
		return
	}
	part := r.URL.Query().Get("part")
	filename, contentType, data, err := h.svc.Download(r.Context(), domain, part)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *CertHandler) Delete(w http.ResponseWriter, r *http.Request) {
	domain := strings.TrimSpace(r.PathValue("domain"))
	if domain == "" {
		writeError(w, http.StatusBadRequest, "域名不能为空")
		return
	}
	if err := h.svc.Delete(r.Context(), domain); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书已删除"})
}

func (h *CertHandler) Renew(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain string `json:"domain"`
		CA     string `json:"ca"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	record, err := h.svc.Renew(r.Context(), req.Domain, req.CA)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}
