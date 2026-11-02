package awareness

import (
	"fmt"
	"time"
)

// Event is a single row shown in the live feed.
type Event struct {
	At       string    `json:"at"`
	ClientIP string    `json:"client_ip"`
	Service  string    `json:"service"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Status   int       `json:"status"`
	Kind     EventKind `json:"kind"`
	Summary  string    `json:"summary"`
	Geo      string    `json:"geo,omitempty"`
}

// TopIP summarizes one client in the sliding window.
type TopIP struct {
	IP       string      `json:"ip"`
	Requests int         `json:"requests"`
	Services []string    `json:"services"`
	Kinds    []EventKind `json:"kinds"`
}

// ServiceRow aggregates today's traffic per proxy rule name.
type ServiceRow struct {
	Service       string `json:"service"`
	Requests      int    `json:"requests"`
	UniqueIPs     int    `json:"unique_ips"`
	Blocked       int    `json:"blocked"`
	ScanSuspected int    `json:"scan_suspected"`
}

// TodayStats holds calendar-day counters in the configured timezone.
type TodayStats struct {
	TotalRequests  int `json:"total_requests"`
	UniqueIPs      int `json:"unique_ips"`
	Safe           int `json:"safe"`
	Blocked403     int `json:"blocked_403"`
	RateLimited429 int `json:"rate_limited_429"`
	AuthFailed401  int `json:"auth_failed_401"`
	BlockedTotal   int `json:"blocked_total"`
	ScanSuspected  int `json:"scan_suspected"`
	AbnormalBurst  int `json:"abnormal_burst"`
}

// LiveSnapshot is returned by GET /api/awareness/live.
type LiveSnapshot struct {
	UpdatedAt         string       `json:"updated_at"`
	Timezone          string       `json:"timezone"`
	Range             string       `json:"range"`
	TrendGranularity  string       `json:"trend_granularity,omitempty"`
	WindowMinutes     int          `json:"window_minutes"`
	Today             TodayStats   `json:"today"`
	RequestYesterday  int          `json:"request_yesterday"`
	RequestTrend      *int         `json:"request_trend,omitempty"`
	HourlyRequests    []int        `json:"hourly_requests"`
	HourlyUniqueIPs   []int        `json:"hourly_unique_ips"`
	HourlyLabels      []string     `json:"hourly_labels"`
	CurrentHourBucket int          `json:"current_hour_bucket"`
	RecentEvents      []Event      `json:"recent_events"`
	TopIPs            []TopIP      `json:"top_ips"`
	ByService         []ServiceRow `json:"by_service"`
}

const windowDuration = 15 * time.Minute
const maxRecentEvents = 30
const maxTopIPs = 10

func calendarHourlyLabels() []string {
	labels := make([]string, 12)
	for i := 0; i < 12; i++ {
		labels[i] = formatHourLabel(i * 2)
	}
	return labels
}

func formatHourLabel(hour int) string {
	return fmt.Sprintf("%02d:00", hour)
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
