package analytics

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/awareness"
	"github.com/fonu/fonu/internal/ingest"
	"github.com/fonu/fonu/internal/settings"
)

type Service struct {
	db        *sql.DB
	settings  *settings.Store
	ingest    *ingest.Service
	serverGeo func(context.Context) ServerNode

	overviewMu    sync.Mutex
	overviewCache map[string]overviewCacheItem
}

type overviewCacheItem struct {
	at time.Time
	ov Overview
}

// 首页会同时打 overview 和 awareness/live，后者内部再算一遍 overview。
// 短缓存把同一次进入合成一次聚合。
const overviewCacheTTL = 3 * time.Second

func New(db *sql.DB, settingsStore *settings.Store, ingestSvc *ingest.Service) *Service {
	return &Service{db: db, settings: settingsStore, ingest: ingestSvc}
}

type Overview struct {
	UpdatedAt      string                  `json:"updated_at"`
	Timezone       string                  `json:"timezone"`
	Range          string                  `json:"range"`
	BackfillActive bool                    `json:"backfill_active"`
	Realtime       ingest.RealtimeSnapshot `json:"realtime"`
	KPI            KPIBlock                `json:"kpi"`
	Series         SeriesBlock             `json:"series"`
	ByProvince     []ProvinceRow           `json:"by_province"`
	ByCountry      []CountryRow            `json:"by_country"`
	RegionVisitors []RegionVisitorRow      `json:"region_visitors"`
	TopIPs         []TopIPRow              `json:"top_ips"`
	Clients        []ClientRow             `json:"clients"`
	StatusCodes    []StatusRow             `json:"status_codes"`
	Funnel         FunnelBlock             `json:"funnel"`
	Server         ServerNode              `json:"server"`
	Anomaly        AnomalyBlock            `json:"anomaly"`
}

// SetServerGeo supplies the machine's own location (public IP → 归属地) for the map hub.
func (s *Service) SetServerGeo(fn func(context.Context) ServerNode) {
	if s == nil {
		return
	}
	s.serverGeo = fn
}

type KPIBlock struct {
	TotalRequests  int   `json:"total_requests"`
	UniqueIPs      int   `json:"unique_ips"`
	RiskRequests   int   `json:"risk_requests"`
	YesterdayTotal int   `json:"yesterday_total"`
	RequestTrend   *int  `json:"request_trend,omitempty"`
	BytesTotal     int64 `json:"bytes_total"`
}

type FunnelBlock struct {
	Safe           int `json:"safe"`
	BlockedTotal   int `json:"blocked_total"`
	Blocked403     int `json:"blocked_403"`
	RateLimited429 int `json:"rate_limited_429"`
	AuthFailed401  int `json:"auth_failed_401"`
	ScanSuspected  int `json:"scan_suspected"`
}

type SeriesBlock struct {
	Labels        []string `json:"labels"`
	Requests      []int    `json:"requests"`
	UniqueIPs     []int    `json:"unique_ips"`
	Bytes         []int64  `json:"bytes"`
	Granularity   string   `json:"granularity"`
	CurrentBucket int      `json:"current_bucket"`
}

type ProvinceRow struct {
	Province  string `json:"province"`
	Requests  int    `json:"requests"`
	Intercept int    `json:"intercept"`
}

type CountryRow struct {
	Country   string `json:"country"`
	Requests  int    `json:"requests"`
	Intercept int    `json:"intercept"`
}

type RegionVisitorRow struct {
	Region   string `json:"region"`
	Visitors int    `json:"visitors"`
}

type TopIPRow struct {
	IP       string `json:"ip"`
	Requests int    `json:"requests"`
	Bytes    int64  `json:"bytes"`
	Geo      string `json:"geo"`
}

type ClientRow struct {
	Name     string `json:"name"`
	Requests int    `json:"requests"`
}

type StatusRow struct {
	Status int `json:"status"`
	Count  int `json:"count"`
}

func (s *Service) Overview(ctx context.Context, rangeKey string) (Overview, error) {
	rangeKey = awareness.NormalizeRange(rangeKey)
	s.overviewMu.Lock()
	defer s.overviewMu.Unlock()
	if item, ok := s.overviewCache[rangeKey]; ok && time.Since(item.at) < overviewCacheTTL {
		return item.ov, nil
	}
	ov, err := s.computeOverview(ctx, rangeKey)
	if err != nil {
		return ov, err
	}
	if s.overviewCache == nil {
		s.overviewCache = make(map[string]overviewCacheItem)
	}
	s.overviewCache[rangeKey] = overviewCacheItem{at: time.Now(), ov: ov}
	return ov, nil
}

func (s *Service) computeOverview(ctx context.Context, rangeKey string) (Overview, error) {
	rangeKey = awareness.NormalizeRange(rangeKey)
	loc, tz := s.location(ctx)
	now := time.Now().In(loc)
	period := awareness.PeriodForRange(rangeKey, now, loc)
	start := period.Start.UTC().Format(time.RFC3339)
	end := period.End.UTC().Format(time.RFC3339)

	excludePrivate, _ := s.settings.GetBool(ctx, settings.KeyAnalyticsExcludePrivateIPs)
	ipFilter := ""
	if excludePrivate {
		ipFilter = ` AND client_ip NOT IN ('127.0.0.1','::1') AND client_ip NOT LIKE '10.%' AND client_ip NOT LIKE '192.168.%' AND client_ip NOT GLOB '172.[16-31].*' `
	}

	out := Overview{
		UpdatedAt: now.UTC().Format(time.RFC3339),
		Timezone:  tz,
		Range:     rangeKey,
	}
	geoDone := make(chan ServerNode, 1)
	if s.serverGeo != nil {
		go func() {
			geoDone <- s.serverGeo(ctx)
		}()
	}
	if s.ingest != nil {
		out.BackfillActive = s.ingest.BackfillActive()
		out.Realtime = s.ingest.Realtime().Snapshot(now)
	}

	kpi, funnel, err := s.queryKPI(ctx, start, end, ipFilter)
	if err != nil {
		return out, err
	}
	out.KPI = kpi
	out.Funnel = funnel

	if rangeKey == awareness.RangeToday {
		yStart := startOfDay(now.AddDate(0, 0, -1), loc).UTC().Format(time.RFC3339)
		yEnd := startOfDay(now, loc).UTC().Format(time.RFC3339)
		yesterday, _ := s.countRequests(ctx, yStart, yEnd, ipFilter)
		out.KPI.YesterdayTotal = yesterday
		out.KPI.RequestTrend = requestTrendPercent(out.KPI.TotalRequests, yesterday)
	}

	series, cur, err := s.querySeries(ctx, period, ipFilter)
	if err != nil {
		return out, err
	}
	out.Series = series
	out.Series.CurrentBucket = cur

	out.ByProvince, _ = s.queryProvinces(ctx, start, end, ipFilter)
	if countries, err := s.queryCountries(ctx, start, end, ipFilter); err == nil {
		out.ByCountry = mergeCountryRows(countries)
	}
	if regions, err := s.queryRegionVisitors(ctx, start, end, ipFilter, 60); err == nil {
		out.RegionVisitors = mergeRegionVisitors(regions, 0)
	}
	out.TopIPs, _ = s.queryTopIPs(ctx, start, end, ipFilter, 15)
	out.Clients, _ = s.queryClients(ctx, start, end, ipFilter)
	out.StatusCodes, _ = s.queryStatus(ctx, start, end, ipFilter)
	if anomaly, err := s.queryAnomaly(ctx, now, loc, ipFilter); err == nil {
		out.Anomaly = anomaly
	}
	if s.serverGeo != nil {
		select {
		case out.Server = <-geoDone:
		case <-ctx.Done():
		}
	}
	return out, nil
}

func (s *Service) MetricsForStatus(ctx context.Context) (today, yesterday int, hourly []int, labels []string, errors int, trend *int) {
	loc, _ := s.location(ctx)
	now := time.Now().In(loc)
	start := startOfDay(now, loc).UTC().Format(time.RFC3339)
	end := now.UTC().Format(time.RFC3339)
	excludePrivate, _ := s.settings.GetBool(ctx, settings.KeyAnalyticsExcludePrivateIPs)
	ipFilter := ""
	if excludePrivate {
		ipFilter = privateIPSQL
	}
	today, _ = s.countRequests(ctx, start, end, ipFilter)
	yStart := startOfDay(now.AddDate(0, 0, -1), loc).UTC().Format(time.RFC3339)
	yEnd := start
	yesterday, _ = s.countRequests(ctx, yStart, yEnd, ipFilter)
	trend = requestTrendPercent(today, yesterday)

	period := awareness.PeriodForRange(awareness.RangeToday, now, loc)
	series, _, _ := s.querySeries(ctx, period, ipFilter)
	hourly = series.Requests
	labels = series.Labels
	kpi, _, _ := s.queryKPI(ctx, start, end, ipFilter)
	errors = kpi.RiskRequests
	return
}

func (s *Service) LiveSnapshot(ctx context.Context, rangeKey string) (awareness.LiveSnapshot, error) {
	ov, err := s.Overview(ctx, rangeKey)
	if err != nil {
		return awareness.LiveSnapshot{}, err
	}
	recent, _ := s.recentEvents(ctx, 30)
	topIPs := make([]awareness.TopIP, 0, len(ov.TopIPs))
	for _, row := range ov.TopIPs {
		topIPs = append(topIPs, awareness.TopIP{IP: row.IP, Requests: row.Requests})
	}
	byService, _ := s.byService(ctx, ov.Range)
	return awareness.LiveSnapshot{
		UpdatedAt:        ov.UpdatedAt,
		Timezone:         ov.Timezone,
		Range:            ov.Range,
		TrendGranularity: ov.Series.Granularity,
		WindowMinutes:    15,
		Today: awareness.TodayStats{
			TotalRequests:  ov.KPI.TotalRequests,
			UniqueIPs:      ov.KPI.UniqueIPs,
			Safe:           ov.Funnel.Safe,
			Blocked403:     ov.Funnel.Blocked403,
			RateLimited429: ov.Funnel.RateLimited429,
			AuthFailed401:  ov.Funnel.AuthFailed401,
			BlockedTotal:   ov.Funnel.BlockedTotal,
			ScanSuspected:  ov.Funnel.ScanSuspected,
		},
		RequestYesterday:  ov.KPI.YesterdayTotal,
		RequestTrend:      ov.KPI.RequestTrend,
		HourlyRequests:    ov.Series.Requests,
		HourlyUniqueIPs:   ov.Series.UniqueIPs,
		HourlyLabels:      ov.Series.Labels,
		CurrentHourBucket: ov.Series.CurrentBucket,
		RecentEvents:      recent,
		TopIPs:            topIPs,
		ByService:         byService,
	}, nil
}

const privateIPSQL = ` AND client_ip NOT IN ('127.0.0.1','::1') AND client_ip NOT LIKE '10.%' AND client_ip NOT LIKE '192.168.%' AND client_ip NOT GLOB '172.[16-31].*' `

func (s *Service) location(ctx context.Context) (*time.Location, string) {
	tz := settings.DefaultTimezone
	if s.settings != nil {
		if raw, _ := s.settings.Get(ctx, settings.KeyTimezone); strings.TrimSpace(raw) != "" {
			tz = strings.TrimSpace(raw)
		}
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
		tz = settings.DefaultTimezone
	}
	return loc, tz
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func requestTrendPercent(today, yesterday int) *int {
	if yesterday <= 0 {
		if today <= 0 {
			return nil
		}
		v := 100
		return &v
	}
	pct := (today - yesterday) * 100 / yesterday
	return &pct
}

func (s *Service) countRequests(ctx context.Context, start, end, ipFilter string) (int, error) {
	q := `SELECT COUNT(*) FROM access_facts WHERE ts >= ? AND ts <= ?` + ipFilter
	var n int
	err := s.db.QueryRowContext(ctx, q, start, end).Scan(&n)
	return n, err
}

func (s *Service) queryKPI(ctx context.Context, start, end, ipFilter string) (KPIBlock, FunnelBlock, error) {
	var kpi KPIBlock
	var funnel FunnelBlock
	q := `
		SELECT
			COUNT(*),
			COUNT(DISTINCT client_ip),
			COALESCE(SUM(bytes_sent), 0),
			SUM(CASE WHEN kind IN ('blocked','rate_limited','auth_failed','scan_suspected','abnormal_burst') THEN 1 ELSE 0 END),
			SUM(CASE WHEN status >= 200 AND status < 400 AND kind = 'access' THEN 1 ELSE 0 END),
			SUM(CASE WHEN kind = 'blocked' OR status = 403 THEN 1 ELSE 0 END),
			SUM(CASE WHEN kind = 'rate_limited' OR status = 429 THEN 1 ELSE 0 END),
			SUM(CASE WHEN kind = 'auth_failed' OR status = 401 THEN 1 ELSE 0 END),
			SUM(CASE WHEN kind = 'scan_suspected' THEN 1 ELSE 0 END)
		FROM access_facts WHERE ts >= ? AND ts <= ?` + ipFilter
	err := s.db.QueryRowContext(ctx, q, start, end).Scan(
		&kpi.TotalRequests,
		&kpi.UniqueIPs,
		&kpi.BytesTotal,
		&kpi.RiskRequests,
		&funnel.Safe,
		&funnel.Blocked403,
		&funnel.RateLimited429,
		&funnel.AuthFailed401,
		&funnel.ScanSuspected,
	)
	funnel.BlockedTotal = funnel.Blocked403 + funnel.RateLimited429 + funnel.AuthFailed401
	return kpi, funnel, err
}
