package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/acme"
	"github.com/fonu/fonu/internal/analytics"
	certstore "github.com/fonu/fonu/internal/certificate"
	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/ddns"
	"github.com/fonu/fonu/internal/nginx"
	"github.com/fonu/fonu/internal/publicip"
	"github.com/fonu/fonu/internal/service"
)

type StatusHandler struct {
	cfg          config.Config
	proxySvc     *service.ProxyService
	ddnsSvc      *ddns.Service
	acmeSvc      *acme.Service
	analytics    *analytics.Service
	startedAt    string
	nginxPIDFile string

	publicIPMu       sync.Mutex
	publicIPv4       string
	publicIPv6       string
	publicIPSource   string
	publicIPCachedAt time.Time
 publicIPFlight chan struct{}
 publicIPDetect func(context.Context)(string,string,error)
 runtimeMu sync.Mutex
 runtimeVersions map[string]string
 runtimeVersionAt time.Time
}

func NewStatusHandler(cfg config.Config, proxySvc *service.ProxyService, ddnsSvc *ddns.Service, acmeSvc *acme.Service, analyticsSvc *analytics.Service, startedAt, nginxPIDFile string) *StatusHandler {
	return &StatusHandler{
		cfg:          cfg,
		proxySvc:     proxySvc,
		ddnsSvc:      ddnsSvc,
		acmeSvc:      acmeSvc,
		analytics:    analyticsSvc,
		startedAt:    startedAt,
		nginxPIDFile: nginxPIDFile,
	}
}

type statusResponse struct {
	PublicIPv4           string   `json:"public_ipv4"`
	PublicIPv6           string   `json:"public_ipv6"`
	PublicIPSource       string   `json:"public_ip_source"`
	DDNSStatus           string   `json:"ddns_status"`
	DDNSCount            int      `json:"ddns_count"`
	DDNSLastUpdated      string   `json:"ddns_last_updated,omitempty"`
	CertificateStatus    string   `json:"certificate_status"`
	CertificateDays      int      `json:"certificate_days"`
	CertificateCount     int      `json:"certificate_count"`
	CertificateSummary   string   `json:"certificate_summary,omitempty"`
	ProxyCount           int      `json:"proxy_count"`
	NginxStatus          string   `json:"nginx_status"`
	NginxAvailable       bool     `json:"nginx_available"`
	NginxBin             string   `json:"nginx_bin,omitempty"`
	RequestToday         int      `json:"request_today"`
	RequestYesterday     int      `json:"request_yesterday"`
	RequestTrend         *int     `json:"request_trend,omitempty"`
	RequestsHourly       []int    `json:"requests_hourly"`
	RequestsHourlyLabels []string `json:"requests_hourly_labels"`
	ErrorToday           int      `json:"error_today"`
	AvgResponseMs        float64  `json:"avg_response_ms"`
	StartedAt            string   `json:"started_at"`
	UptimeSeconds        int64    `json:"uptime_seconds"`
}

func (h *StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	rules, err := h.proxySvc.List(r.Context())
	if err != nil {
		writeError(r, w, http.StatusInternalServerError, "读取状态失败")
		return
	}

	nginxStatus := "unknown"
	if _, err := os.Stat(h.nginxPIDFile); err == nil {
		nginxStatus = "running"
	} else if os.IsNotExist(err) {
		nginxStatus = "stopped"
	}

	ddnsStatus, ddnsLastUpdated, ddnsCount := h.ddnsSvc.Summary(r.Context())
	ipv4, ipv6, ipSource := h.resolvePublicIPs(r.Context())

	records, _ := h.acmeSvc.List(r.Context())
	certStatus, certDays, certCount, certSummary := summarizeCertificates(records)

	// 访问量由 /api/analytics/overview 提供。这里再扫一遍会让首页每次进入多等一轮聚合。
	hourly := []int{}
	hourlyLabels := []string{}

	started, _ := time.Parse(time.RFC3339, h.startedAt)
	uptime := time.Since(started).Seconds()

	writeJSON(w, http.StatusOK, statusResponse{
		PublicIPv4:           ipv4,
		PublicIPv6:           ipv6,
		PublicIPSource:       ipSource,
		DDNSStatus:           ddnsStatus,
		DDNSCount:            ddnsCount,
		DDNSLastUpdated:      ddnsLastUpdated,
		CertificateStatus:    certStatus,
		CertificateDays:      certDays,
		CertificateCount:     certCount,
		CertificateSummary:   certSummary,
		ProxyCount:           len(rules),
		NginxStatus:          nginxStatus,
		NginxAvailable:       nginx.BinAvailable(h.cfg.NginxBin),
		NginxBin:             h.cfg.NginxBin,
		RequestsHourly:       hourly,
		RequestsHourlyLabels: hourlyLabels,
		StartedAt:            h.startedAt,
		UptimeSeconds:        int64(uptime),
	})
}

func(h *StatusHandler) resolvePublicIPs(ctx context.Context)(ipv4,ipv6,source string){
 if h.ddnsSvc!=nil{
  ipv4,ipv6,_=h.ddnsSvc.PublicIPs(ctx)
  if ipv4==""||ipv6==""{v4,v6,_:=h.ddnsSvc.LastKnownPublicIPs(ctx);if ipv4==""{ipv4=v4};if ipv6==""{ipv6=v6}}
 }
 if ipv4!=""&&ipv6!=""{return ipv4,ipv6,"ddns"}
 source="none";if ipv4!=""||ipv6!=""{source="ddns"}
 // Prefer existing DDNS information; fill the missing family from a cached
 // independent exit probe, with a short deadline for the dashboard.
 detectCtx,cancel:=context.WithTimeout(ctx,1500*time.Millisecond);defer cancel()
 v4,v6,_:=h.detectPublicIPs(detectCtx,false)
 filled:=false
 if ipv4==""&&v4!=""{ipv4=v4;filled=true}
 if ipv6==""&&v6!=""{ipv6=v6;filled=true}
 if filled{if source=="ddns"{source="mixed"}else{source="detect"}}
 return
}
func(h *StatusHandler) detectPublicIPs(ctx context.Context,force bool)(v4,v6 string,checked time.Time){
 h.publicIPMu.Lock()
 if !force&&h.publicIPFlight==nil&&!h.publicIPCachedAt.IsZero()&&time.Since(h.publicIPCachedAt)<time.Minute{
  v4,v6,checked=h.publicIPv4,h.publicIPv6,h.publicIPCachedAt;h.publicIPMu.Unlock();return
 }
 flight:=h.publicIPFlight
 if flight==nil{
  flight=make(chan struct{});h.publicIPFlight=flight
  detect:=h.publicIPDetect;if detect==nil{detect=publicip.Detect}
  // Dashboard callers have a short budget, but a shared probe can finish
  // independently for DDNS and subsequent requests. At most one is active.
  go func(done chan struct{}){
   probeCtx,cancel:=context.WithTimeout(context.Background(),6*time.Second);defer cancel()
   a,b,_:=detect(probeCtx)
   h.publicIPMu.Lock()
   h.publicIPv4,h.publicIPv6=a,b;h.publicIPCachedAt=time.Now()
   h.publicIPFlight=nil;close(done);h.publicIPMu.Unlock()
  }(flight)
 }
 h.publicIPMu.Unlock()
 select{case <-flight:case <-ctx.Done():}
 h.publicIPMu.Lock();v4,v6,checked=h.publicIPv4,h.publicIPv6,h.publicIPCachedAt;h.publicIPMu.Unlock()
 return
}
func(h *StatusHandler) PublicIP(w http.ResponseWriter,r *http.Request){
 v4,v6,checked:=h.detectPublicIPs(r.Context(),r.URL.Query().Get("refresh")=="1")
 w.Header().Set("Cache-Control","no-store")
 at:="";if !checked.IsZero(){at=checked.UTC().Format(time.RFC3339)}
 writeJSON(w,http.StatusOK,map[string]string{"public_ipv4":v4,"public_ipv6":v6,"checked_at":at})
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
			status = "expiring"
		}
	}
	if count == 1 {
		summary = certificateSummaryName(records[0])
	} else {
		summary = fmt.Sprintf("%d 张证书", count)
	}
	return status, days, count, summary
}

func requestTrendPercent(today, yesterday int) *int {
	if yesterday <= 0 {
		if today <= 0 {
			return nil
		}
		v := 100
		return &v
	}
	v := int(math.Round(float64(today-yesterday) / float64(yesterday) * 100))
	return &v
}

func certificateSummaryName(rec certstore.Record) string {
	for _, domain := range rec.Domains {
		if strings.HasPrefix(domain, "*.") {
			return domain
		}
	}
	if rec.Wildcard {
		return "*." + rec.Domain
	}
	return rec.Domain
}
