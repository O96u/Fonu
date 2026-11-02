package awareness

import (
	"bufio"
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/proxy"
	"github.com/fonu/fonu/internal/settings"
)

type RuleLister func(ctx context.Context) ([]proxy.Rule, error)

type Analyzer struct {
	path      string
	settings  *settings.Store
	listRules RuleLister

	mu sync.RWMutex

	timezone       string
	loc            *time.Location
	dayKey         string
	yesterdayTotal int

	bindings proxy.BindingIndex

	today        TodayStats
	todayIPs     map[string]struct{}
	hourlyReq    [12]int
	hourlyIPSets [12]map[string]struct{}
	serviceStats map[string]*serviceAgg

	windowStart time.Time
	windowIPs   map[string]*windowIPState
	recent      []timedEvent

	updatedAt time.Time

	scanCacheRange string
	scanCacheAt    time.Time
	scanCacheSnap  LiveSnapshot
}

type serviceAgg struct {
	ips      map[string]struct{}
	requests int
	blocked  int
	scan     int
}

type windowIPState struct {
	requests int
	services map[string]struct{}
	kinds    map[EventKind]struct{}
}

type timedEvent struct {
	at    time.Time
	event Event
}

func NewAnalyzer(logPath string, settingsStore *settings.Store, listRules RuleLister) *Analyzer {
	return &Analyzer{
		path:         logPath,
		settings:     settingsStore,
		listRules:    listRules,
		todayIPs:     map[string]struct{}{},
		serviceStats: map[string]*serviceAgg{},
		windowIPs:    map[string]*windowIPState{},
		bindings:     proxy.BindingIndex{},
	}
}

func (a *Analyzer) Start(ctx context.Context) {
	a.refreshMeta(ctx)
	go func() {
		a.bootstrapToday(ctx)
		go a.tailLoop(ctx)
	}()
	go a.metaLoop(ctx)
}

func (a *Analyzer) metaLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refreshMeta(ctx)
		}
	}
}

func (a *Analyzer) refreshMeta(ctx context.Context) {
	tz := settings.DefaultTimezone
	if a.settings != nil {
		if raw, _ := a.settings.Get(ctx, settings.KeyTimezone); strings.TrimSpace(raw) != "" {
			tz = strings.TrimSpace(raw)
		}
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
		tz = "Asia/Shanghai"
	}
	rules, _ := a.listRules(ctx)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.timezone = tz
	a.loc = loc
	a.bindings = proxy.BuildBindingIndex(rules)
	now := time.Now().In(loc)
	key := now.Format("2006-01-02")
	if a.dayKey != "" && a.dayKey != key {
		a.resetTodayLocked(key)
	} else if a.dayKey == "" {
		a.dayKey = key
	}
}

func (a *Analyzer) bootstrapToday(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	a.refreshMeta(ctx)
	a.mu.RLock()
	loc := a.loc
	if loc == nil {
		loc = time.Local
	}
	a.mu.RUnlock()

	start := startOfDay(time.Now().In(loc), loc)
	lines, err := logstore.ReadAccessLinesSince(a.path, start, 0)
	if err != nil {
		return
	}
	for _, line := range lines {
		if ctx.Err() != nil {
			return
		}
		a.ingestLine(line, true)
	}
	a.mu.Lock()
	a.updatedAt = time.Now()
	a.mu.Unlock()
}

func (a *Analyzer) tailLoop(ctx context.Context) {
	offset := a.logOffset()
	for {
		if ctx.Err() != nil {
			return
		}
		offset = a.readFrom(offset)
		time.Sleep(500 * time.Millisecond)
	}
}

func (a *Analyzer) logOffset() int64 {
	info, err := os.Stat(a.path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (a *Analyzer) readFrom(offset int64) int64 {
	file, err := os.Open(a.path)
	if err != nil {
		return offset
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return offset
	}
	if info.Size() < offset {
		offset = 0
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return offset
	}
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			a.ingestLine(strings.TrimRight(line, "\r\n"), true)
		}
		if err != nil {
			if err == io.EOF {
				pos, seekErr := file.Seek(0, io.SeekCurrent)
				if seekErr == nil {
					return pos
				}
				return info.Size()
			}
			return offset
		}
	}
}

func (a *Analyzer) ingestLine(line string, trackWindow bool) {
	entry, ok := logstore.ParseAccess(line)
	if !ok || !logstore.IsDashboardAccess(entry) {
		return
	}
	at, ok := parseAccessTime(entry.Time)
	if !ok {
		at = time.Now()
	}

	a.mu.RLock()
	loc := a.loc
	if loc == nil {
		loc = time.Local
	}
	a.mu.RUnlock()

	at = at.In(loc)
	now := time.Now().In(loc)
	if !sameCalendarDay(at, now) {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.loc == nil {
		a.loc = time.Local
	}
	dayKey := now.Format("2006-01-02")
	if a.dayKey != dayKey {
		a.resetTodayLocked(dayKey)
	}

	service := proxy.ResolveServiceLabel(entry.Domain, entry.ServerPort, entry.Upstream, a.bindings)
	ip := strings.TrimSpace(entry.ClientIP)
	if ip == "" {
		ip = "—"
	}

	if sameCalendarDay(at, now) {
		a.recordTodayLocked(entry, service, ip, at)
	}

	if trackWindow {
		a.recordWindowLocked(entry, service, ip, at, now)
	}
	a.updatedAt = now
}

func (a *Analyzer) recordTodayLocked(entry logstore.AccessEntry, service, ip string, at time.Time) {
	a.today.TotalRequests++
	a.todayIPs[ip] = struct{}{}
	a.today.UniqueIPs = len(a.todayIPs)

	bucket := at.Hour() / 2
	if bucket >= 0 && bucket < 12 {
		a.hourlyReq[bucket]++
		if a.hourlyIPSets[bucket] == nil {
			a.hourlyIPSets[bucket] = map[string]struct{}{}
		}
		a.hourlyIPSets[bucket][ip] = struct{}{}
	}

	switch entry.Status {
	case 403:
		a.today.Blocked403++
		a.today.BlockedTotal++
	case 429:
		a.today.RateLimited429++
		a.today.BlockedTotal++
	case 401:
		a.today.AuthFailed401++
		a.today.BlockedTotal++
	default:
		if entry.Status >= 200 && entry.Status < 400 {
			a.today.Safe++
		}
	}

	st := a.serviceStats[service]
	if st == nil {
		st = &serviceAgg{ips: map[string]struct{}{}}
		a.serviceStats[service] = st
	}
	st.requests++
	st.ips[ip] = struct{}{}

	ipWin := &ipWindowState{}
	kind, _ := classifyEntry(entry.Status, entry.Path, entry.UserAgent, ipWin)
	if kind == KindScanSuspected {
		a.today.ScanSuspected++
		st.scan++
	}
	if kind == KindAbnormalBurst {
		a.today.AbnormalBurst++
	}
	if entry.Status == 403 || entry.Status == 429 || entry.Status == 401 {
		st.blocked++
	}
}

func (a *Analyzer) recordWindowLocked(entry logstore.AccessEntry, service, ip string, at, now time.Time) {
	cutoff := now.Add(-windowDuration)
	if at.Before(cutoff) {
		return
	}

	w := a.windowIPs[ip]
	if w == nil {
		w = &windowIPState{services: map[string]struct{}{}, kinds: map[EventKind]struct{}{}}
		a.windowIPs[ip] = w
	}
	w.requests++
	w.services[service] = struct{}{}

	ipWin := &ipWindowState{total: w.requests}
	if entry.Status == 404 {
		ipWin.notFound = 1
	}
	ipWin.notFound += countRecentIPStatus(a.recent, ip, 404, cutoff)
	ipWin.blocked = countRecentIPStatus(a.recent, ip, 0, cutoff)

	kind, _ := classifyEntry(entry.Status, entry.Path, entry.UserAgent, ipWin)
	w.kinds[kind] = struct{}{}

	ev := Event{
		At:       at.UTC().Format(time.RFC3339),
		ClientIP: ip,
		Service:  service,
		Domain:   entry.Domain,
		Path:     entry.Path,
		Status:   entry.Status,
		Kind:     kind,
		Summary:  eventSummary(kind, entry.Status, entry.Path),
	}
	a.recent = append([]timedEvent{{at: at, event: ev}}, a.recent...)
	if len(a.recent) > maxRecentEvents*4 {
		a.recent = a.recent[:maxRecentEvents*4]
	}

	var kept []timedEvent
	for _, te := range a.recent {
		if !te.at.Before(cutoff) {
			kept = append(kept, te)
		}
	}
	a.recent = kept
	a.compactWindowLocked(cutoff)
}

func countRecentIPStatus(recent []timedEvent, ip string, status int, cutoff time.Time) int {
	n := 0
	for _, te := range recent {
		if te.at.Before(cutoff) || te.event.ClientIP != ip {
			continue
		}
		if status == 0 {
			switch te.event.Status {
			case 401, 403, 429:
				n++
			}
			continue
		}
		if te.event.Status == status {
			n++
		}
	}
	return n
}

func (a *Analyzer) compactWindowLocked(cutoff time.Time) {
	next := map[string]*windowIPState{}
	for _, te := range a.recent {
		if te.at.Before(cutoff) {
			continue
		}
		ip := te.event.ClientIP
		w := next[ip]
		if w == nil {
			w = &windowIPState{services: map[string]struct{}{}, kinds: map[EventKind]struct{}{}}
			next[ip] = w
		}
		w.requests++
		w.services[te.event.Service] = struct{}{}
		w.kinds[te.event.Kind] = struct{}{}
	}
	a.windowIPs = next
	a.windowStart = cutoff
}

func (a *Analyzer) resetTodayLocked(dayKey string) {
	a.yesterdayTotal = a.today.TotalRequests
	a.dayKey = dayKey
	a.today = TodayStats{}
	a.todayIPs = map[string]struct{}{}
	a.hourlyReq = [12]int{}
	a.hourlyIPSets = [12]map[string]struct{}{}
	a.serviceStats = map[string]*serviceAgg{}
}

func sameCalendarDay(t, day time.Time) bool {
	y1, m1, d1 := t.Date()
	y2, m2, d2 := day.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func parseAccessTime(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

const scanCacheTTL = 45 * time.Second

// LiveForRange returns stats for today, 7d, 30d, or all (file scan with short cache).
func (a *Analyzer) LiveForRange(rangeKey string) LiveSnapshot {
	rangeKey = NormalizeRange(rangeKey)
	a.mu.RLock()
	if rangeKey == RangeToday {
		snap := a.buildTodaySnapshotLocked()
		a.mu.RUnlock()
		return snap
	}
	if a.scanCacheRange == rangeKey && time.Since(a.scanCacheAt) < scanCacheTTL {
		snap := a.scanCacheSnap
		a.mu.RUnlock()
		return snap
	}
	loc := a.loc
	tz := a.timezone
	bindings := a.bindings
	a.mu.RUnlock()

	if loc == nil {
		loc = time.Local
	}
	if tz == "" {
		tz = settings.DefaultTimezone
	}
	period := PeriodForRange(rangeKey, time.Now(), loc)
	snap := a.aggregatePeriod(period, bindings)
	snap.Timezone = tz

	a.mu.Lock()
	a.scanCacheRange = rangeKey
	a.scanCacheAt = time.Now()
	a.scanCacheSnap = snap
	a.mu.Unlock()
	return snap
}

func (a *Analyzer) buildTodaySnapshotLocked() LiveSnapshot {
	now := time.Now()
	if a.loc != nil {
		now = now.In(a.loc)
	}
	currentBucket := now.Hour() / 2
	if currentBucket >= 12 {
		currentBucket = 11
	}

	hourlyUnique := make([]int, 12)
	for i := 0; i < 12; i++ {
		if a.hourlyIPSets[i] != nil {
			hourlyUnique[i] = len(a.hourlyIPSets[i])
		}
	}

	recent := TailRecentEvents(a.path, a.bindings, maxRecentEvents)

	updated := a.updatedAt
	if updated.IsZero() {
		updated = time.Now()
	}

	tz := a.timezone
	if tz == "" {
		tz = settings.DefaultTimezone
	}

	reqSlice := make([]int, 12)
	copy(reqSlice, a.hourlyReq[:])

	return LiveSnapshot{
		UpdatedAt:         updated.UTC().Format(time.RFC3339),
		Timezone:          tz,
		Range:             RangeToday,
		TrendGranularity:  "two_hour",
		WindowMinutes:     int(windowDuration / time.Minute),
		Today:             a.today,
		RequestYesterday:  a.yesterdayTotal,
		RequestTrend:      requestTrendPercent(a.today.TotalRequests, a.yesterdayTotal),
		HourlyRequests:    reqSlice,
		HourlyUniqueIPs:   hourlyUnique,
		HourlyLabels:      calendarHourlyLabels(),
		CurrentHourBucket: currentBucket,
		RecentEvents:      recent,
		TopIPs:            a.buildTopIPsLocked(),
		ByService:         a.buildByServiceLocked(),
	}
}

// Live returns today's snapshot (same as LiveForRange("today")).
func (a *Analyzer) Live() LiveSnapshot {
	return a.LiveForRange(RangeToday)
}

func (a *Analyzer) buildTopIPsLocked() []TopIP {
	type row struct {
		ip   string
		data *windowIPState
	}
	rows := make([]row, 0, len(a.windowIPs))
	for ip, data := range a.windowIPs {
		rows = append(rows, row{ip: ip, data: data})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].data.requests > rows[j].data.requests
	})
	limit := maxTopIPs
	if len(rows) < limit {
		limit = len(rows)
	}
	out := make([]TopIP, 0, limit)
	for i := 0; i < limit; i++ {
		services := make([]string, 0, len(rows[i].data.services))
		for s := range rows[i].data.services {
			services = append(services, s)
		}
		sort.Strings(services)
		kinds := make([]EventKind, 0, len(rows[i].data.kinds))
		for k := range rows[i].data.kinds {
			kinds = append(kinds, k)
		}
		sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
		out = append(out, TopIP{
			IP:       rows[i].ip,
			Requests: rows[i].data.requests,
			Services: services,
			Kinds:    kinds,
		})
	}
	return out
}

func (a *Analyzer) buildByServiceLocked() []ServiceRow {
	rows := make([]ServiceRow, 0, len(a.serviceStats))
	for name, st := range a.serviceStats {
		rows = append(rows, ServiceRow{
			Service:       name,
			Requests:      st.requests,
			UniqueIPs:     len(st.ips),
			Blocked:       st.blocked,
			ScanSuspected: st.scan,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Requests == rows[j].Requests {
			return rows[i].Service < rows[j].Service
		}
		return rows[i].Requests > rows[j].Requests
	})
	if len(rows) > 20 {
		rows = rows[:20]
	}
	return rows
}

// MetricsForStatus exposes today counters for /api/status without scanning logs.
func (a *Analyzer) MetricsForStatus() (today, yesterday int, hourly []int, labels []string, errors int, trend *int) {
	snap := a.Live()
	today = snap.Today.TotalRequests
	yesterday = snap.RequestYesterday
	hourly = snap.HourlyRequests
	labels = snap.HourlyLabels
	trend = snap.RequestTrend
	errors = snap.Today.BlockedTotal + snap.Today.ScanSuspected
	return
}
