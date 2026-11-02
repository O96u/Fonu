package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/fonu/fonu/internal/awareness"
)

func (s *Service) querySeries(ctx context.Context, period awareness.Period, ipFilter string) (SeriesBlock, int, error) {
	out := SeriesBlock{Granularity: period.BucketKind}
	if period.BucketKind == "two_hour" {
		out.Labels = make([]string, 12)
		out.Requests = make([]int, 12)
		out.UniqueIPs = make([]int, 12)
		out.Bytes = make([]int64, 12)
		for i := 0; i < 12; i++ {
			out.Labels[i] = fmt.Sprintf("%02d:00", i*2)
		}
		rows, err := s.db.QueryContext(ctx, `
			SELECT CAST(strftime('%H', ts, 'localtime') AS INTEGER) / 2 AS bucket,
			       COUNT(*), COUNT(DISTINCT client_ip), COALESCE(SUM(bytes_sent),0)
			FROM access_facts
			WHERE ts >= ? AND ts <= ?`+ipFilter+`
			GROUP BY bucket
		`, period.Start.UTC().Format(time.RFC3339), period.End.UTC().Format(time.RFC3339))
		if err != nil {
			return out, 0, err
		}
		defer rows.Close()
		for rows.Next() {
			var bucket, reqs, ips int
			var bytes int64
			if err := rows.Scan(&bucket, &reqs, &ips, &bytes); err != nil {
				return out, 0, err
			}
			if bucket >= 0 && bucket < 12 {
				out.Requests[bucket] = reqs
				out.UniqueIPs[bucket] = ips
				out.Bytes[bucket] = bytes
			}
		}
		cur := period.End.Hour() / 2
		if cur >= 12 {
			cur = 11
		}
		return out, cur, rows.Err()
	}

	out.Labels = make([]string, 0, period.MaxBuckets)
	out.Requests = make([]int, 0, period.MaxBuckets)
	out.UniqueIPs = make([]int, 0, period.MaxBuckets)
	labelIndex := map[string]int{}
	for d := period.Start; !d.After(period.End) && len(out.Labels) < period.MaxBuckets; d = d.AddDate(0, 0, 1) {
		label := d.Format("01-02")
		labelIndex[d.Format("2006-01-02")] = len(out.Labels)
		out.Labels = append(out.Labels, label)
		out.Requests = append(out.Requests, 0)
		out.UniqueIPs = append(out.UniqueIPs, 0)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT date(ts, 'localtime') AS day, COUNT(*), COUNT(DISTINCT client_ip)
		FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		GROUP BY day
	`, period.Start.UTC().Format(time.RFC3339), period.End.UTC().Format(time.RFC3339))
	if err != nil {
		return out, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var day string
		var reqs, ips int
		if err := rows.Scan(&day, &reqs, &ips); err != nil {
			return out, 0, err
		}
		if idx, ok := labelIndex[day]; ok {
			out.Requests[idx] = reqs
			out.UniqueIPs[idx] = ips
		}
	}
	out.Granularity = "daily"
	return out, len(out.Labels) - 1, rows.Err()
}

func (s *Service) queryProvinces(ctx context.Context, start, end, ipFilter string) ([]ProvinceRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT CASE WHEN geo_country = '中国' AND geo_province != '' THEN geo_province
		            WHEN geo_country != '' AND geo_country != '中国' THEN '海外'
		            ELSE '未知' END AS province,
		       COUNT(*) AS c,
		       SUM(CASE WHEN kind IN ('blocked','rate_limited','auth_failed','scan_suspected','abnormal_burst')
		             OR status IN (401,403,429) THEN 1 ELSE 0 END) AS intercept
		FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		GROUP BY province
		ORDER BY c DESC
		LIMIT 40
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProvinceRow
	for rows.Next() {
		var row ProvinceRow
		if err := rows.Scan(&row.Province, &row.Requests, &row.Intercept); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) queryCountries(ctx context.Context, start, end, ipFilter string) ([]CountryRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT CASE WHEN geo_country != '' THEN geo_country ELSE '未知' END AS country,
		       COUNT(*) AS c,
		       SUM(CASE WHEN kind IN ('blocked','rate_limited','auth_failed','scan_suspected','abnormal_burst')
		             OR status IN (401,403,429) THEN 1 ELSE 0 END) AS intercept
		FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		GROUP BY country
		ORDER BY c DESC
		LIMIT 80
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CountryRow
	for rows.Next() {
		var row CountryRow
		if err := rows.Scan(&row.Country, &row.Requests, &row.Intercept); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) queryRegionVisitors(ctx context.Context, start, end, ipFilter string, limit int) ([]RegionVisitorRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT CASE WHEN geo_country = '中国' AND geo_province != '' THEN geo_province
		            WHEN geo_country != '' AND geo_country != '中国' THEN geo_country
		            ELSE '未知' END AS region,
		       COUNT(DISTINCT client_ip) AS visitors
		FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		GROUP BY region
		HAVING region != '未知'
		ORDER BY visitors DESC
		LIMIT ?
	`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RegionVisitorRow
	for rows.Next() {
		var row RegionVisitorRow
		if err := rows.Scan(&row.Region, &row.Visitors); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) queryTopIPs(ctx context.Context, start, end, ipFilter string, limit int) ([]TopIPRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT client_ip, COUNT(*), COALESCE(SUM(bytes_sent),0), MAX(geo_label)
		FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		GROUP BY client_ip
		ORDER BY COUNT(*) DESC
		LIMIT ?
	`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TopIPRow
	for rows.Next() {
		var row TopIPRow
		if err := rows.Scan(&row.IP, &row.Requests, &row.Bytes, &row.Geo); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) queryClients(ctx context.Context, start, end, ipFilter string) ([]ClientRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_agent, COUNT(*) FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		GROUP BY user_agent
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var ua string
		var c int
		if err := rows.Scan(&ua, &c); err != nil {
			return nil, err
		}
		name := classifyUA(ua)
		counts[name] += c
	}
	out := make([]ClientRow, 0, len(counts))
	for name, c := range counts {
		out = append(out, ClientRow{Name: name, Requests: c})
	}
	sortClientRows(out)
	if len(out) > 8 {
		out = out[:8]
	}
	return out, rows.Err()
}

func (s *Service) queryStatus(ctx context.Context, start, end, ipFilter string) ([]StatusRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		GROUP BY status
		ORDER BY COUNT(*) DESC
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatusRow
	for rows.Next() {
		var row StatusRow
		if err := rows.Scan(&row.Status, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) recentEvents(ctx context.Context, limit int) ([]awareness.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ts, client_ip, service, host, path, status, kind, geo_label
		FROM access_facts
		ORDER BY ts DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []awareness.Event
	for rows.Next() {
		var ev awareness.Event
		var kind string
		if err := rows.Scan(&ev.At, &ev.ClientIP, &ev.Service, &ev.Domain, &ev.Path, &ev.Status, &kind, &ev.Geo); err != nil {
			return nil, err
		}
		ev.Kind = awareness.EventKind(kind)
		ev.Summary = eventSummary(ev.Kind, ev.Status, ev.Path)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *Service) byService(ctx context.Context, rangeKey string) ([]awareness.ServiceRow, error) {
	loc, _ := s.location(ctx)
	now := time.Now().In(loc)
	period := awareness.PeriodForRange(rangeKey, now, loc)
	start := period.Start.UTC().Format(time.RFC3339)
	end := period.End.UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT service,
		       COUNT(*),
		       COUNT(DISTINCT client_ip),
		       SUM(CASE WHEN kind IN ('blocked','rate_limited','auth_failed') OR status IN (401,403,429) THEN 1 ELSE 0 END),
		       SUM(CASE WHEN kind = 'scan_suspected' THEN 1 ELSE 0 END)
		FROM access_facts
		WHERE ts >= ? AND ts <= ?
		GROUP BY service
		ORDER BY COUNT(*) DESC
		LIMIT 20
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []awareness.ServiceRow
	for rows.Next() {
		var row awareness.ServiceRow
		if err := rows.Scan(&row.Service, &row.Requests, &row.UniqueIPs, &row.Blocked, &row.ScanSuspected); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func eventSummary(kind awareness.EventKind, status int, path string) string {
	switch kind {
	case awareness.KindBlocked:
		return "策略拦截（403）"
	case awareness.KindRateLimited:
		return "触发限速（429）"
	case awareness.KindAuthFailed:
		return "认证失败（401）"
	case awareness.KindScanSuspected:
		return "疑似扫描"
	case awareness.KindAbnormalBurst:
		return "高频异常访问"
	default:
		if status >= 500 {
			return "服务端错误"
		}
		return "正常访问"
	}
}
