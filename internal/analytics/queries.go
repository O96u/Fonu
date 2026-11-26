package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/fonu/fonu/internal/awareness"
)

// Bucket boundaries are generated in the configured application timezone,
// not SQLite's host-local timezone. Range scans preserve the ts index.
func (s *Service) querySeries(ctx context.Context, period awareness.Period, ipFilter string) (SeriesBlock, int, error) {
	out := SeriesBlock{Granularity: period.BucketKind}
	// Use persisted UTC-minute rollups for complete minutes; raw facts remain
	// authoritative for partial boundary minutes and exact distinct visitors.
	var rollupExists int
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'access_minute_ip'").Scan(&rollupExists)
	loc := period.Start.Location()
	type bucket struct { start, end time.Time; label string }
	var buckets []bucket
	if period.BucketKind == "two_hour" {
		y, m, d := period.Start.Date()
		for i := 0; i < 12; i++ {
			buckets = append(buckets, bucket{time.Date(y,m,d,i*2,0,0,0,loc),time.Date(y,m,d,(i+1)*2,0,0,0,loc),fmt.Sprintf("%02d:00",i*2)})
		}
	} else {
		out.Granularity = "daily"
		for d := period.Start; !d.After(period.End) && len(buckets)<period.MaxBuckets; d=d.AddDate(0,0,1) {
			buckets=append(buckets,bucket{d,d.AddDate(0,0,1),d.Format("01-02")})
		}
	}
 // Avoid hundreds of empty index/rollup queries on new installations or
 // when log retention is shorter than the selected year.
 var earliest,latest string
 rangeStart:=period.Start.UTC().Format(time.RFC3339)
 rangeEnd:=period.End.UTC().Format(time.RFC3339)
 err:=s.db.QueryRowContext(ctx,`SELECT COALESCE((SELECT ts FROM access_facts WHERE ts>=? AND ts<=? ORDER BY ts ASC LIMIT 1),''),COALESCE((SELECT ts FROM access_facts WHERE ts>=? AND ts<=? ORDER BY ts DESC LIMIT 1),'')`,rangeStart,rangeEnd,rangeStart,rangeEnd).Scan(&earliest,&latest)
 if err!=nil{return out,-1,err}
 current := -1
 for i, b := range buckets {
		out.Labels=append(out.Labels,b.label)
		out.Requests=append(out.Requests,0)
		out.UniqueIPs=append(out.UniqueIPs,0)
		out.Bytes=append(out.Bytes,0)
		if b.start.After(period.End) { continue }
		current=i
		end := b.end
		if cap := period.End.Add(time.Second); cap.Before(end) { end=cap }
		if !end.After(b.start) { continue }
  if earliest==""||b.start.UTC().Format(time.RFC3339)>latest||end.UTC().Format(time.RFC3339)<=earliest{continue}
		var err error
		if rollupExists > 0 {
			err = s.querySeriesBucket(ctx,b.start,end,ipFilter,&out.Requests[i],&out.UniqueIPs[i],&out.Bytes[i])
		} else {
			err = s.db.QueryRowContext(ctx,`SELECT COUNT(*), COUNT(DISTINCT client_ip), COALESCE(SUM(bytes_sent),0) FROM access_facts WHERE ts >= ? AND ts < ?`+ipFilter,b.start.UTC().Format(time.RFC3339),end.UTC().Format(time.RFC3339)).Scan(&out.Requests[i],&out.UniqueIPs[i],&out.Bytes[i])
		}
		if err != nil { return out, current, err }
	}
	return out,current,nil
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

func (s *Service) recentEvents(ctx context.Context, start,end,ipFilter string, limit int) ([]awareness.Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ts, client_ip, service, host, path, status, kind, geo_label
		FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
		ORDER BY ts DESC, id DESC
		LIMIT ?
	`, start,end,limit)
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

func (s *Service) byService(ctx context.Context, start,end,ipFilter string) ([]awareness.ServiceRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT service,
		       COUNT(*),
		       COUNT(DISTINCT client_ip),
		       SUM(CASE WHEN kind IN ('blocked','rate_limited','auth_failed') OR status IN (401,403,429) THEN 1 ELSE 0 END),
		       SUM(CASE WHEN kind = 'scan_suspected' THEN 1 ELSE 0 END)
		FROM access_facts
		WHERE ts >= ? AND ts <= ?`+ipFilter+`
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

func (s *Service) querySeriesBucket(ctx context.Context, start,end time.Time, ipFilter string, requests,ips *int, bytes *int64) error {
	// All facts are stored at whole-second precision.
	start,end=start.UTC().Truncate(time.Second),end.UTC().Truncate(time.Second)
	first:=start.Truncate(time.Minute)
	if first.Before(start) { first=first.Add(time.Minute) }
	last:=end.Truncate(time.Minute)
	if !first.Before(last) {
		return s.db.QueryRowContext(ctx,`SELECT COUNT(*),COUNT(DISTINCT client_ip),COALESCE(SUM(bytes_sent),0) FROM access_facts WHERE ts>=? AND ts<?`+ipFilter,start.Format(time.RFC3339),end.Format(time.RFC3339)).Scan(requests,ips,bytes)
	}
	q:=`SELECT COALESCE(SUM(requests),0),COUNT(DISTINCT client_ip),COALESCE(SUM(bytes),0) FROM (
		SELECT client_ip,requests,bytes FROM access_minute_ip WHERE minute>=? AND minute<?`+ipFilter+`
		UNION ALL SELECT client_ip,1,bytes_sent FROM access_facts WHERE ts>=? AND ts<?`+ipFilter+`
		UNION ALL SELECT client_ip,1,bytes_sent FROM access_facts WHERE ts>=? AND ts<?`+ipFilter+`)`
	return s.db.QueryRowContext(ctx,q,first.Format(time.RFC3339),last.Format(time.RFC3339),start.Format(time.RFC3339),first.Format(time.RFC3339),last.Format(time.RFC3339),end.Format(time.RFC3339)).Scan(requests,ips,bytes)
}
