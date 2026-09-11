package api

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/ddns"
	"github.com/fonu/fonu/internal/acme"
	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/publicip"
	"github.com/fonu/fonu/internal/service"
)

type StatusHandler struct {
	cfg          config.Config
	proxySvc     *service.ProxyService
	ddnsSvc      *ddns.Service
	acmeSvc      *acme.Service
	startedAt    string
	nginxPIDFile string
}

func NewStatusHandler(cfg config.Config, proxySvc *service.ProxyService, ddnsSvc *ddns.Service, acmeSvc *acme.Service, startedAt, nginxPIDFile string) *StatusHandler {
	return &StatusHandler{
		cfg:          cfg,
		proxySvc:     proxySvc,
		ddnsSvc:      ddnsSvc,
		acmeSvc:      acmeSvc,
		startedAt:    startedAt,
		nginxPIDFile: nginxPIDFile,
	}
}

type statusResponse struct {
	PublicIPv4        string  `json:"public_ipv4"`
	PublicIPv6        string  `json:"public_ipv6"`
	DDNSStatus        string  `json:"ddns_status"`
	DDNSLastUpdated   string  `json:"ddns_last_updated,omitempty"`
	CertificateStatus string  `json:"certificate_status"`
	CertificateDays   int     `json:"certificate_days"`
	ProxyCount        int     `json:"proxy_count"`
	NginxStatus       string  `json:"nginx_status"`
	RequestToday      int     `json:"request_today"`
	ErrorToday        int     `json:"error_today"`
	AvgResponseMs     float64 `json:"avg_response_ms"`
	StartedAt         string  `json:"started_at"`
	UptimeSeconds     int64   `json:"uptime_seconds"`
}

func (h *StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	rules, err := h.proxySvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取状态失败")
		return
	}

	nginxStatus := "unknown"
	if _, err := os.Stat(h.nginxPIDFile); err == nil {
		nginxStatus = "running"
	} else if os.IsNotExist(err) {
		nginxStatus = "stopped"
	}

	ipv4, ipv6, _ := publicip.Detect(r.Context())

	ddnsStatus := "disabled"
	ddnsLastUpdated := ""
	if cfg, err := h.ddnsSvc.Get(r.Context()); err == nil {
		ddnsStatus = cfg.LastStatus
		if cfg.LastUpdatedAt != nil {
			ddnsLastUpdated = cfg.LastUpdatedAt.Format(time.RFC3339)
		}
		if !cfg.Enabled {
			ddnsStatus = "disabled"
		}
	}

	certStatus := "none"
	certDays := 0
	if records, err := h.acmeSvc.List(r.Context()); err == nil && len(records) > 0 {
		rec := records[0]
		certStatus = rec.Status
		certDays = rec.DaysLeft
		if certDays <= 30 && certDays > 0 {
			certStatus = "warning"
		}
		if certDays <= 0 && rec.Status == "ok" {
			certStatus = "error"
		}
	}

	total, errors, avgMs := logstore.CountTodayAccess(filepath.Join(h.cfg.LogsDir(), "access.log"))

	started, _ := time.Parse(time.RFC3339, h.startedAt)
	uptime := time.Since(started).Seconds()

	writeJSON(w, http.StatusOK, statusResponse{
		PublicIPv4:        ipv4,
		PublicIPv6:        ipv6,
		DDNSStatus:        ddnsStatus,
		DDNSLastUpdated:   ddnsLastUpdated,
		CertificateStatus: certStatus,
		CertificateDays:   certDays,
		ProxyCount:        len(rules),
		NginxStatus:       nginxStatus,
		RequestToday:      total,
		ErrorToday:        errors,
		AvgResponseMs:     avgMs,
		StartedAt:         h.startedAt,
		UptimeSeconds:     int64(uptime),
	})
}
