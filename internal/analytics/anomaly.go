package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

const (
	anomalySlowMS     = 1000
	anomalyIssueAvgMS = 500
	anomalyIssueLimit = 5
	anomalyTrendDays  = 7
)

// AnomalyMetric is today compared with yesterday.
type AnomalyMetric struct {
	Current    float64  `json:"current"`
	Previous   float64  `json:"previous"`
	Delta      float64  `json:"delta"`
	ChangeRate *float64 `json:"change_rate,omitempty"`
	Share      *float64 `json:"share,omitempty"`
}

// AnomalyPoint is one day on the 7-day error and latency chart.
type AnomalyPoint struct {
	Label        string  `json:"label"`
	Status4xx    int     `json:"status_4xx"`
	Status5xx    int     `json:"status_5xx"`
	AvgRequestMS float64 `json:"avg_request_ms"`
}

// AnomalyURL is a path that produced 5xx, slow requests, or high latency.
type AnomalyURL struct {
	Path              string  `json:"path"`
	Requests          int     `json:"requests"`
	Errors5xx         int     `json:"errors_5xx"`
	Errors5xxDelta    int     `json:"errors_5xx_delta"`
	Slow              int     `json:"slow"`
	AvgRequestMS      float64 `json:"avg_request_ms"`
	AvgRequestDeltaMS float64 `json:"avg_request_delta_ms"`
}

// AnomalyBlock is the homepage anomaly insight. Upstream time and request
// body size are not in access_facts, so they are omitted.
type AnomalyBlock struct {
	SlowThresholdMS int            `json:"slow_threshold_ms"`
	TotalRequests   int            `json:"total_requests"`
	Status5xx       AnomalyMetric  `json:"status_5xx"`
	Status4xx       AnomalyMetric  `json:"status_4xx"`
	AvgRequestMS    AnomalyMetric  `json:"avg_request_ms"`
	SlowRate        AnomalyMetric  `json:"slow_rate"`
	Trend           []AnomalyPoint `json:"trend"`
	TopURLs         []AnomalyURL   `json:"top_urls"`
}

type anomalyDay struct {
	total        int
	status4xx    int
	status5xx    int
	slow         int
	latencyCount int
	avgMS        float64
}

// 耗时只统计真正返回的短请求。101 的 $request_time 是 WebSocket 整段连接时间，
// path 为 "-" 的是没有请求行的超时或坏请求，放进平均值会把今天拉到几万毫秒。
const anomalyLatencyWhere = `status != 101 AND path != '-'`

func (s *Service) queryAnomaly(ctx context.Context, now time.Time, loc *time.Location, ipFilter string) (AnomalyBlock, error) {
	out := AnomalyBlock{
		SlowThresholdMS: anomalySlowMS,
		Trend:           make([]AnomalyPoint, 0, anomalyTrendDays),
		TopURLs:         []AnomalyURL{},
	}
	todayStart := startOfDay(now, loc)
	trendStart := todayStart.AddDate(0, 0, -(anomalyTrendDays - 1))
	days, err := s.queryAnomalyDays(ctx, trendStart, now, ipFilter)
	if err != nil {
		return out, err
	}

	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	var today, yesterday anomalyDay
	for i := 0; i < anomalyTrendDays; i++ {
		day := trendStart.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		snap := days[key]
		out.Trend = append(out.Trend, AnomalyPoint{
			Label:        fmt.Sprintf("%s %s", day.Format("01-02"), weekdays[day.Weekday()]),
			Status4xx:    snap.status4xx,
			Status5xx:    snap.status5xx,
			AvgRequestMS: snap.avgMS,
		})
		if key == todayStart.Format("2006-01-02") {
			today = snap
		}
		if key == todayStart.AddDate(0, 0, -1).Format("2006-01-02") {
			yesterday = snap
		}
	}

	out.TotalRequests = today.total
	out.Status5xx = anomalyCountMetric(today.status5xx, yesterday.status5xx, today.total)
	out.Status4xx = anomalyCountMetric(today.status4xx, yesterday.status4xx, today.total)
	out.AvgRequestMS = anomalyFloatMetric(today.avgMS, yesterday.avgMS)
	out.SlowRate = anomalyFloatMetric(anomalyRate(today.slow, today.latencyCount), anomalyRate(yesterday.slow, yesterday.latencyCount))

	urls, err := s.queryAnomalyURLs(ctx, todayStart, now, todayStart.AddDate(0, 0, -1), todayStart, ipFilter)
	if err != nil {
		return out, err
	}
	out.TopURLs = urls
	return out, nil
}

func (s *Service) queryAnomalyDays(ctx context.Context, start, end time.Time, ipFilter string) (map[string]anomalyDay, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT date(ts, 'localtime') AS day,
		       COUNT(*) AS total,
		       SUM(CASE WHEN status >= 400 AND status < 500 THEN 1 ELSE 0 END),
		       SUM(CASE WHEN status >= 500 AND status < 600 THEN 1 ELSE 0 END),
		       SUM(CASE WHEN `+anomalyLatencyWhere+` AND request_ms >= ? THEN 1 ELSE 0 END),
		       AVG(CASE WHEN `+anomalyLatencyWhere+` THEN request_ms END),
		       SUM(CASE WHEN `+anomalyLatencyWhere+` THEN 1 ELSE 0 END)
		FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		GROUP BY day
	`, anomalySlowMS, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]anomalyDay{}
	for rows.Next() {
		var day string
		var snap anomalyDay
		var avg sql.NullFloat64
		if err := rows.Scan(&day, &snap.total, &snap.status4xx, &snap.status5xx, &snap.slow, &avg, &snap.latencyCount); err != nil {
			return nil, err
		}
		if avg.Valid {
			snap.avgMS = avg.Float64
		}
		out[day] = snap
	}
	return out, rows.Err()
}

func (s *Service) queryAnomalyURLs(ctx context.Context, curStart, curEnd, prevStart, prevEnd time.Time, ipFilter string) ([]AnomalyURL, error) {
	pathExpr := `CASE WHEN instr(path, '?') > 0 THEN substr(path, 1, instr(path, '?') - 1) ELSE path END`
	query := `
		SELECT ` + pathExpr + ` AS p,
		       COUNT(*) AS requests,
		       SUM(CASE WHEN status >= 500 AND status < 600 THEN 1 ELSE 0 END) AS errors_5xx,
		       SUM(CASE WHEN request_ms >= ? THEN 1 ELSE 0 END) AS slow_requests,
		       AVG(request_ms) AS avg_ms
		FROM access_facts
		WHERE ts >= ? AND ts <= ? AND path != '' AND ` + anomalyLatencyWhere + ipFilter + `
		GROUP BY p
		HAVING errors_5xx > 0 OR slow_requests > 0 OR COALESCE(avg_ms, 0) >= ?
		ORDER BY errors_5xx DESC, slow_requests DESC, avg_ms DESC, requests DESC
		LIMIT ?`
	cur, err := scanAnomalyURLs(ctx, s.db, query, anomalySlowMS, curStart.UTC().Format(time.RFC3339), curEnd.UTC().Format(time.RFC3339), anomalyIssueAvgMS, anomalyIssueLimit)
	if err != nil {
		return nil, err
	}
	if len(cur) == 0 {
		return []AnomalyURL{}, nil
	}
	prev, err := scanAnomalyURLs(ctx, s.db, `
		SELECT `+pathExpr+` AS p,
		       COUNT(*) AS requests,
		       SUM(CASE WHEN status >= 500 AND status < 600 THEN 1 ELSE 0 END) AS errors_5xx,
		       SUM(CASE WHEN request_ms >= ? THEN 1 ELSE 0 END) AS slow_requests,
		       AVG(request_ms) AS avg_ms
		FROM access_facts
		WHERE ts >= ? AND ts < ? AND path != '' AND `+anomalyLatencyWhere+ipFilter+`
		GROUP BY p
	`, anomalySlowMS, prevStart.UTC().Format(time.RFC3339), prevEnd.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	prevByPath := map[string]anomalyURLRow{}
	for _, row := range prev {
		prevByPath[row.path] = row
	}
	sort.SliceStable(cur, func(i, j int) bool {
		if cur[i].errors5xx != cur[j].errors5xx {
			return cur[i].errors5xx > cur[j].errors5xx
		}
		if cur[i].slow != cur[j].slow {
			return cur[i].slow > cur[j].slow
		}
		if cur[i].avgMS != cur[j].avgMS {
			return cur[i].avgMS > cur[j].avgMS
		}
		return cur[i].requests > cur[j].requests
	})
	out := make([]AnomalyURL, 0, len(cur))
	for _, row := range cur {
		old := prevByPath[row.path]
		out = append(out, AnomalyURL{
			Path:              row.path,
			Requests:          row.requests,
			Errors5xx:         row.errors5xx,
			Errors5xxDelta:    row.errors5xx - old.errors5xx,
			Slow:              row.slow,
			AvgRequestMS:      row.avgMS,
			AvgRequestDeltaMS: row.avgMS - old.avgMS,
		})
	}
	return out, nil
}

type anomalyURLRow struct {
	path      string
	requests  int
	errors5xx int
	slow      int
	avgMS     float64
}

func scanAnomalyURLs(ctx context.Context, db *sql.DB, query string, args ...any) ([]anomalyURLRow, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []anomalyURLRow
	for rows.Next() {
		var row anomalyURLRow
		var avg sql.NullFloat64
		if err := rows.Scan(&row.path, &row.requests, &row.errors5xx, &row.slow, &avg); err != nil {
			return nil, err
		}
		if avg.Valid {
			row.avgMS = avg.Float64
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func anomalyCountMetric(current, previous, total int) AnomalyMetric {
	metric := anomalyFloatMetric(float64(current), float64(previous))
	if total > 0 {
		share := float64(current) * 100 / float64(total)
		metric.Share = &share
	}
	return metric
}

func anomalyFloatMetric(current, previous float64) AnomalyMetric {
	metric := AnomalyMetric{Current: current, Previous: previous, Delta: current - previous}
	if previous == 0 {
		if current == 0 {
			zero := 0.0
			metric.ChangeRate = &zero
		}
		return metric
	}
	rate := (current - previous) * 100 / previous
	metric.ChangeRate = &rate
	return metric
}

func anomalyRate(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(part) * 100 / float64(total)
}
