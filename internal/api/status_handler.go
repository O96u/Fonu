package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fonu/fonu/internal/acme"
	certstore "github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/ddns"
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
	DDNSCount         int     `json:"ddns_count"`
	DDNSLastUpdated   string  `json:"ddns_last_updated,omitempty"`
	CertificateStatus  string `json:"certificate_status"`
	CertificateDays    int    `json:"certificate_days"`
	CertificateCount   int    `json:"certificate_count"`
	CertificateSummary string `json:"certificate_summary,omitempty"`
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

	ddnsStatus, ddnsLastUpdated, ddnsCount := h.ddnsSvc.Summary(r.Context())

	records, _ := h.acmeSvc.List(r.Context())
	certStatus, certDays, certCount, certSummary := summarizeCertificates(records)

	total, errors, avgMs := logstore.CountTodayAccess(filepath.Join(h.cfg.LogsDir(), "access.log"))

	started, _ := time.Parse(time.RFC3339, h.startedAt)
	uptime := time.Since(started).Seconds()

	writeJSON(w, http.StatusOK, statusResponse{
		PublicIPv4: ipv4,
		PublicIPv6:        ipv6,
		DDNSStatus:        ddnsStatus,
		DDNSCount:         ddnsCount,
		DDNSLastUpdated:   ddnsLastUpdated,
		CertificateStatus:  certStatus,
		CertificateDays:    certDays,
		CertificateCount:   certCount,
		CertificateSummary: certSummary,
		ProxyCount:        len(rules),
		NginxStatus:       nginxStatus,
		RequestToday:      total,
		ErrorToday:        errors,
		AvgResponseMs:     avgMs,
		StartedAt:         h.startedAt,
		UptimeSeconds: int64(uptime),
	})
}

func summarizeCertificates(records []certstore.Record) (status string, days int, count int, summary string) {
	if len(records) == 0 {
		return "none", 0, 0, ""
	}
	count = len(records)
	status = "ok"
	days = records[0].DaysLeft
	for _, rec := range records {
		if rec.DaysLeft < days {
			days = rec.DaysLeft
		}
		switch {
		case rec.Status == "error" || rec.DaysLeft <= 0:
			status = "error"
		case rec.DaysLeft <= 30 && status != "error":
			status = "warning"
		}
	}
	if count == 1 {
		summary = "*." + records[0].Domain
	} else {
		summary = fmt.Sprintf("%d 张证书", count)
	}
	return status, days, count, summary
}
