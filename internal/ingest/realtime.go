package ingest

import (
	"sync"
	"time"
)

// RealtimeRing aggregates access log records by second. Memory is bounded
// by the time window rather than by the number of incoming requests.
type RealtimeRing struct {
	window time.Duration
	mu sync.Mutex
	samples map[int64]realtimeSample
	lastSweep int64
}

type realtimeSample struct {
	requests int
	bytes int64
	sumMS float64
	msCount int
}

type RealtimeSnapshot struct {
	QPS           float64 `json:"qps"`
	TrafficKBps   float64 `json:"traffic_kbps"`
	AvgUpstreamMS float64 `json:"avg_upstream_ms"`
	WindowSec     int     `json:"window_sec"`
}

func NewRealtimeRing(window time.Duration) *RealtimeRing {
	if window <= 0 { window = 60 * time.Second }
	return &RealtimeRing{window:window, samples:make(map[int64]realtimeSample)}
}

func (r *RealtimeRing) Add(row FactRow) { r.addAt(row, time.Now()) }

func (r *RealtimeRing) addAt(row FactRow, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lastSweep != now.Unix() { r.pruneLocked(now); r.lastSweep=now.Unix() }
	if row.TS.Before(now.Add(-r.window)) || row.TS.After(now) { return }
	key := row.TS.Unix()
	s := r.samples[key]
	s.requests++
	s.bytes += row.BytesSent
	if row.RequestMS > 0 { s.sumMS += row.RequestMS; s.msCount++ }
	r.samples[key] = s
}

func (r *RealtimeRing) Snapshot(now time.Time) RealtimeSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pruneLocked(now)
	r.lastSweep = now.Unix()
	var reqs, msCount int
	var bytes int64
	var sumMS float64
	for _, s := range r.samples {
		reqs += s.requests
		bytes += s.bytes
		sumMS += s.sumMS
		msCount += s.msCount
	}
	span := r.window.Seconds()
	if span < 1 { span = 1 }
	out := RealtimeSnapshot{
		QPS: float64(reqs)/span,
		TrafficKBps: float64(bytes)/span/1024,
		WindowSec: int(span),
	}
	if msCount > 0 { out.AvgUpstreamMS = sumMS/float64(msCount) }
	return out
}

func (r *RealtimeRing) pruneLocked(now time.Time) {
	// At most one second of boundary precision is lost: access log records
	// use second-resolution timestamps. Also discard future buckets on clock rollback.
	cutoff := now.Add(-r.window).Unix()
	for at := range r.samples {
		if at < cutoff || at > now.Unix() { delete(r.samples,at) }
	}
}
