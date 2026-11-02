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

	overviewMu     sync.Mutex
	overviewCache  map[string]overviewCacheItem
	overviewFlight map[string]*overviewCall
	eventGeo       func([]awareness.Event)
}

type overviewCacheItem struct {
	at time.Time
	ov Overview
}

// 首页每 5 秒刷新一次。缓存盖住这一轮，同一时间只算一份。
const overviewCacheTTL = 15 * time.Second

type overviewCall struct {
	done chan struct{}
	ov   Overview
	err  error
}

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
	RecentEvents   []awareness.Event       `json:"recent_events"`
	ByService      []awareness.ServiceRow  `json:"by_service"`
}

// SetServerGeo supplies the machine's own location (public IP → 归属地) for the map hub.
func (s *Service) SetServerGeo(fn func(context.Context) ServerNode) {
	if s == nil {
		return
	}
	s.serverGeo = fn
}

// SetEventGeo fills recent-event locations while the overview is built.
func (s *Service) SetEventGeo(fn func([]awareness.Event)) {
	if s == nil {
		return
	}
	s.eventGeo = fn
}

type KPIBlock struct {
	TotalRequests      int   `json:"total_requests"`
	UniqueIPs          int   `json:"unique_ips"`
	RiskRequests       int   `json:"risk_requests"`
	YesterdayTotal     int   `json:"yesterday_total"`
	RequestTrend       *int  `json:"request_trend,omitempty"`
	YesterdayUniqueIPs int   `json:"yesterday_unique_ips"`
	UniqueIPTrend      *int  `json:"unique_ip_trend,omitempty"`
	YesterdayRisk      int   `json:"yesterday_risk"`
	RiskTrend          *int  `json:"risk_trend,omitempty"`
	BytesTotal         int64 `json:"bytes_total"`
}

type FunnelBlock struct {
	Safe           int  `json:"safe"`
	BlockedTotal   int  `json:"blocked_total"`
	Blocked403     int  `json:"blocked_403"`
	RateLimited429 int  `json:"rate_limited_429"`
	AuthFailed401  int  `json:"auth_failed_401"`
	ScanSuspected  int  `json:"scan_suspected"`
	YesterdayScan  int  `json:"yesterday_scan"`
	ScanTrend      *int `json:"scan_trend,omitempty"`
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
	if item, ok := s.overviewCache[rangeKey]; ok && time.Since(item.at) < overviewCacheTTL {
		s.overviewMu.Unlock()
		return item.ov, nil
	}
	if call, ok := s.overviewFlight[rangeKey]; ok {
		s.overviewMu.Unlock()
		select {
		case <-call.done:
			return call.ov, call.err
		case <-ctx.Done():
			return Overview{}, ctx.Err()
		}
	}
	call := &overviewCall{done: make(chan struct{})}
	if s.overviewFlight == nil {
		s.overviewFlight = map[string]*overviewCall{}
	}
	s.overviewFlight[rangeKey] = call
	s.overviewMu.Unlock()

	ov, err := s.computeOverview(ctx, rangeKey)

	s.overviewMu.Lock()
	if err == nil {
		if s.overviewCache == nil {
			s.overviewCache = map[string]overviewCacheItem{}
		}
		s.overviewCache[rangeKey] = overviewCacheItem{at: time.Now(), ov: ov}
	}
	call.ov, call.err = ov, err
	delete(s.overviewFlight, rangeKey)
	close(call.done)
	s.overviewMu.Unlock()
	return ov, err
}

func (s *Service) computeOverview(ctx context.Context, rangeKey string) (Overview, error) {
	rangeKey = awareness.NormalizeRange(rangeKey)
	loc, tz := s.location(ctx)
	now := time.Now().In(loc)
	period := awareness.PeriodForRange(rangeKey, now, loc)
	start := period.Start.UTC().Format(time.RFC3339)
	end := period.End.UTC().Format(time.RFC3339)

	excludePrivate := false
	if s.settings != nil {
		excludePrivate, _ = s.settings.GetBool(ctx, settings.KeyAnalyticsExcludePrivateIPs)
	}
	ipFilter := ""
	if excludePrivate {
		ipFilter = ` AND client_ip NOT IN ('127.0.0.1','::1') AND client_ip NOT LIKE '10.%' AND client_ip NOT LIKE '192.168.%' AND client_ip NOT GLOB '172.[16-31].*' `
	}

	out := Overview{
		UpdatedAt:    now.UTC().Format(time.RFC3339),
		Timezone:     tz,
		Range:        rangeKey,
		RecentEvents: []awareness.Event{},
		ByService:    []awareness.ServiceRow{},
	}
	if s.ingest != nil {
		out.BackfillActive = s.ingest.BackfillActive()
		out.Realtime = s.ingest.Realtime().Snapshot(now)
	}

	var (
		kpi       KPIBlock
		funnel    FunnelBlock
		kpiErr    error
		series    SeriesBlock
		seriesAt  int
		seriesErr error
		provinces []ProvinceRow
		countries []CountryRow
		regions   []RegionVisitorRow
		topIPs    []TopIPRow
		clients   []ClientRow
		statuses  []StatusRow
		anomaly   AnomalyBlock
		anomalyOK bool
		recent    []awareness.Event
		services  []awareness.ServiceRow
		yKPI      KPIBlock
		yFunnel   FunnelBlock
		server    ServerNode
	)
	var wg sync.WaitGroup
	spawn := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}
	spawn(func() { kpi, funnel, kpiErr = s.queryKPI(ctx, start, end, ipFilter) })
	spawn(func() { series, seriesAt, seriesErr = s.querySeries(ctx, period, ipFilter) })
	spawn(func() { provinces, _ = s.queryProvinces(ctx, start, end, ipFilter) })
	spawn(func() { countries, _ = s.queryCountries(ctx, start, end, ipFilter) })
	spawn(func() { regions, _ = s.queryRegionVisitors(ctx, start, end, ipFilter, 60) })
	spawn(func() { topIPs, _ = s.queryTopIPs(ctx, start, end, ipFilter, 15) })
	spawn(func() { clients, _ = s.queryClients(ctx, start, end, ipFilter) })
	spawn(func() { statuses, _ = s.queryStatus(ctx, start, end, ipFilter) })
	spawn(func() {
		var err error
		anomaly, err = s.queryAnomaly(ctx, now, loc, ipFilter)
		anomalyOK = err == nil
	})
	spawn(func() { recent, _ = s.recentEvents(ctx, 30) })
	spawn(func() { services, _ = s.byService(ctx, rangeKey) })
	if rangeKey == awareness.RangeToday {
		yStart := startOfDay(now.AddDate(0, 0, -1), loc).UTC().Format(time.RFC3339)
		yEnd := startOfDay(now, loc).UTC().Format(time.RFC3339)
		spawn(func() { yKPI, yFunnel, _ = s.queryKPI(ctx, yStart, yEnd, ipFilter) })
	}
	if s.serverGeo != nil {
		spawn(func() { server = s.serverGeo(ctx) })
	}
	wg.Wait()
	if kpiErr != nil {
		return out, kpiErr
	}
	if seriesErr != nil {
		return out, seriesErr
	}

	out.KPI = kpi
	out.Funnel = funnel
	if rangeKey == awareness.RangeToday {
		out.KPI.YesterdayTotal = yKPI.TotalRequests
		out.KPI.RequestTrend = requestTrendPercent(out.KPI.TotalRequests, yKPI.TotalRequests)
		out.KPI.YesterdayUniqueIPs = yKPI.UniqueIPs
		out.KPI.UniqueIPTrend = requestTrendPercent(out.KPI.UniqueIPs, yKPI.UniqueIPs)
		out.KPI.YesterdayRisk = yKPI.RiskRequests
		out.KPI.RiskTrend = requestTrendPercent(out.KPI.RiskRequests, yKPI.RiskRequests)
		out.Funnel.YesterdayScan = yFunnel.ScanSuspected
		out.Funnel.ScanTrend = requestTrendPercent(out.Funnel.ScanSuspected, yFunnel.ScanSuspected)
	}
	out.Series = series
	out.Series.CurrentBucket = seriesAt
	out.ByProvince = provinces
	if countries != nil {
		out.ByCountry = mergeCountryRows(countries)
	}
	if regions != nil {
		out.RegionVisitors = mergeRegionVisitors(regions, 0)
	}
	out.TopIPs = topIPs
	out.Clients = clients
	out.StatusCodes = statuses
	if anomalyOK {
		out.Anomaly = anomaly
	}
	if len(recent) > 0 {
		if s.eventGeo != nil {
			s.eventGeo(recent)
		}
		out.RecentEvents = recent
	}
	if len(services) > 0 {
		out.ByService = services
	}
	out.Server = server
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
	topIPs := make([]awareness.TopIP, 0, len(ov.TopIPs))
	for _, row := range ov.TopIPs {
		topIPs = append(topIPs, awareness.TopIP{IP: row.IP, Requests: row.Requests})
	}
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
		RecentEvents:      ov.RecentEvents,
		TopIPs:            topIPs,
		ByService:         ov.ByService,
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
			COALESCE(SUM(CASE WHEN kind IN ('blocked','rate_limited','auth_failed','scan_suspected','abnormal_burst') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status >= 200 AND status < 400 AND kind = 'access' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN kind = 'blocked' OR status = 403 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN kind = 'rate_limited' OR status = 429 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN kind = 'auth_failed' OR status = 401 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN kind = 'scan_suspected' THEN 1 ELSE 0 END), 0)
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
