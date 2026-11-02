package awareness

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/proxy"
)

type aggregateState struct {
	stats        TodayStats
	uniqueIPs    map[string]struct{}
	serviceStats map[string]*serviceAgg
	ipStats      map[string]*ipScanState

	bucketReq     []int
	bucketIPSets  []map[string]struct{}
	bucketLabels  []string
	currentBucket int
}

type ipScanState struct {
	total    int
	notFound int
	blocked  int
	services map[string]struct{}
	kinds    map[EventKind]struct{}
}

func (a *Analyzer) aggregatePeriod(period Period, bindings proxy.BindingIndex) LiveSnapshot {
	st := newAggregateState(period)
	file, err := os.Open(a.path)
	if err != nil {
		return a.emptyPeriodSnapshot(period)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		st.ingestLine(scanner.Text(), period, bindings)
	}
	snap := st.snapshot(period, a.timezone)
	snap.RecentEvents = TailRecentEvents(a.path, bindings, maxRecentEvents)
	return snap
}

func newAggregateState(period Period) *aggregateState {
	labels := make([]string, 0, period.MaxBuckets)
	if period.BucketKind == "two_hour" {
		for i := 0; i < 12; i++ {
			labels = append(labels, formatHourLabel(i*2))
		}
	} else {
		for d := period.Start; !d.After(period.End) && len(labels) < period.MaxBuckets; d = d.AddDate(0, 0, 1) {
			labels = append(labels, d.Format("01-02"))
		}
	}
	return &aggregateState{
		uniqueIPs:    map[string]struct{}{},
		serviceStats: map[string]*serviceAgg{},
		ipStats:      map[string]*ipScanState{},
		bucketReq:    make([]int, len(labels)),
		bucketIPSets: make([]map[string]struct{}, len(labels)),
		bucketLabels: labels,
	}
}

func (st *aggregateState) ingestLine(line string, period Period, bindings proxy.BindingIndex) {
	entry, ok := logstore.ParseAccess(line)
	if !ok || !logstore.IsDashboardAccess(entry) {
		return
	}
	at, ok := parseAccessTime(entry.Time)
	if !ok {
		return
	}
	loc := period.Start.Location()
	at = at.In(loc)
	end := period.End.In(loc)
	if at.Before(period.Start) || at.After(end) {
		return
	}

	service := proxy.ResolveServiceLabel(entry.Domain, entry.ServerPort, entry.Upstream, bindings)
	ip := strings.TrimSpace(entry.ClientIP)
	if ip == "" {
		ip = "—"
	}

	st.stats.TotalRequests++
	st.uniqueIPs[ip] = struct{}{}
	st.stats.UniqueIPs = len(st.uniqueIPs)

	bucket := st.bucketIndex(period, at)
	if bucket >= 0 && bucket < len(st.bucketReq) {
		st.bucketReq[bucket]++
		if st.bucketIPSets[bucket] == nil {
			st.bucketIPSets[bucket] = map[string]struct{}{}
		}
		st.bucketIPSets[bucket][ip] = struct{}{}
		st.currentBucket = bucket
	}

	switch entry.Status {
	case 403:
		st.stats.Blocked403++
		st.stats.BlockedTotal++
	case 429:
		st.stats.RateLimited429++
		st.stats.BlockedTotal++
	case 401:
		st.stats.AuthFailed401++
		st.stats.BlockedTotal++
	default:
		if entry.Status >= 200 && entry.Status < 400 {
			st.stats.Safe++
		}
	}

	svc := st.serviceStats[service]
	if svc == nil {
		svc = &serviceAgg{ips: map[string]struct{}{}}
		st.serviceStats[service] = svc
	}
	svc.requests++
	svc.ips[ip] = struct{}{}

	ipSt := st.ipStats[ip]
	if ipSt == nil {
		ipSt = &ipScanState{services: map[string]struct{}{}, kinds: map[EventKind]struct{}{}}
		st.ipStats[ip] = ipSt
	}
	ipSt.total++
	ipSt.services[service] = struct{}{}
	if entry.Status == 404 {
		ipSt.notFound++
	}
	if entry.Status == 401 || entry.Status == 403 || entry.Status == 429 {
		ipSt.blocked++
	}

	ipWin := &ipWindowState{total: ipSt.total, notFound: ipSt.notFound, blocked: ipSt.blocked}
	kind, _ := classifyEntry(entry.Status, entry.Path, entry.UserAgent, ipWin)
	ipSt.kinds[kind] = struct{}{}
	if kind == KindScanSuspected {
		st.stats.ScanSuspected++
		svc.scan++
	}
	if kind == KindAbnormalBurst {
		st.stats.AbnormalBurst++
	}
	if entry.Status == 401 || entry.Status == 403 || entry.Status == 429 {
		svc.blocked++
	}
}

func (st *aggregateState) bucketIndex(period Period, at time.Time) int {
	if period.BucketKind == "two_hour" {
		b := at.Hour() / 2
		if b >= len(st.bucketReq) {
			return len(st.bucketReq) - 1
		}
		return b
	}
	idx := dayIndex(period.Start, at, at.Location())
	if idx < 0 || idx >= len(st.bucketReq) {
		return -1
	}
	return idx
}

func (st *aggregateState) snapshot(period Period, tz string) LiveSnapshot {
	hourlyUnique := make([]int, len(st.bucketLabels))
	for i := range st.bucketLabels {
		if st.bucketIPSets[i] != nil {
			hourlyUnique[i] = len(st.bucketIPSets[i])
		}
	}
	current := st.currentBucket
	if period.BucketKind == "two_hour" {
		current = period.End.Hour() / 2
		if current >= len(st.bucketReq) {
			current = len(st.bucketReq) - 1
		}
	} else if len(st.bucketLabels) > 0 {
		current = len(st.bucketLabels) - 1
	}

	return LiveSnapshot{
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
		Timezone:          tz,
		Range:             period.Range,
		TrendGranularity:  period.BucketKind,
		WindowMinutes:     int(windowDuration / time.Minute),
		Today:             st.stats,
		HourlyRequests:    append([]int(nil), st.bucketReq...),
		HourlyUniqueIPs:   hourlyUnique,
		HourlyLabels:      append([]string(nil), st.bucketLabels...),
		CurrentHourBucket: current,
		TopIPs:            st.buildTopIPs(),
		ByService:         st.buildByService(),
	}
}

func (st *aggregateState) buildTopIPs() []TopIP {
	rows := make([]TopIP, 0, len(st.ipStats))
	for ip, data := range st.ipStats {
		services := make([]string, 0, len(data.services))
		for s := range data.services {
			services = append(services, s)
		}
		sort.Strings(services)
		kinds := make([]EventKind, 0, len(data.kinds))
		for k := range data.kinds {
			kinds = append(kinds, k)
		}
		sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
		rows = append(rows, TopIP{
			IP:       ip,
			Requests: data.total,
			Services: services,
			Kinds:    kinds,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Requests == rows[j].Requests {
			return rows[i].IP < rows[j].IP
		}
		return rows[i].Requests > rows[j].Requests
	})
	if len(rows) > maxTopIPs {
		rows = rows[:maxTopIPs]
	}
	return rows
}

func (st *aggregateState) buildByService() []ServiceRow {
	rows := make([]ServiceRow, 0, len(st.serviceStats))
	for name, svc := range st.serviceStats {
		rows = append(rows, ServiceRow{
			Service:       name,
			Requests:      svc.requests,
			UniqueIPs:     len(svc.ips),
			Blocked:       svc.blocked,
			ScanSuspected: svc.scan,
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

func (a *Analyzer) emptyPeriodSnapshot(period Period) LiveSnapshot {
	st := newAggregateState(period)
	snap := st.snapshot(period, a.timezone)
	a.mu.RLock()
	bindings := a.bindings
	path := a.path
	a.mu.RUnlock()
	snap.RecentEvents = TailRecentEvents(path, bindings, maxRecentEvents)
	return snap
}
