package ingest

import (
	"sync"
	"time"
)

type RealtimeRing struct {
	window time.Duration
	mu     sync.Mutex
	samples []realtimeSample
}

type realtimeSample struct {
	at        time.Time
	requests  int
	bytes     int64
	requestMS float64
}

type RealtimeSnapshot struct {
	QPS           float64 `json:"qps"`
	TrafficKBps   float64 `json:"traffic_kbps"`
	AvgUpstreamMS float64 `json:"avg_upstream_ms"`
	WindowSec     int     `json:"window_sec"`
}

func NewRealtimeRing(window time.Duration) *RealtimeRing {
	if window <= 0 {
		window = 60 * time.Second
	}
	return &RealtimeRing{window: window}
}

func (r *RealtimeRing) Add(row FactRow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.samples = append(r.samples, realtimeSample{
		at:        row.TS,
		requests:  1,
		bytes:     row.BytesSent,
		requestMS: row.RequestMS,
	})
	r.pruneLocked(time.Now())
}

func (r *RealtimeRing) Snapshot(now time.Time) RealtimeSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pruneLocked(now)
	cutoff := now.Add(-r.window)
	var reqs int
	var bytes int64
	var sumMS float64
	var msCount int
	for _, s := range r.samples {
		if s.at.Before(cutoff) {
			continue
		}
		reqs += s.requests
		bytes += s.bytes
		if s.requestMS > 0 {
			sumMS += s.requestMS
			msCount++
		}
	}
	span := r.window.Seconds()
	if span < 1 {
		span = 1
	}
	out := RealtimeSnapshot{
		QPS:         float64(reqs) / span,
		TrafficKBps: float64(bytes) / span / 1024,
		WindowSec:   int(span),
	}
	if msCount > 0 {
		out.AvgUpstreamMS = sumMS / float64(msCount)
	}
	return out
}

func (r *RealtimeRing) pruneLocked(now time.Time) {
	cutoff := now.Add(-r.window)
	i := 0
	for i < len(r.samples) && r.samples[i].at.Before(cutoff) {
		i++
	}
	if i > 0 {
		r.samples = append([]realtimeSample(nil), r.samples[i:]...)
	}
}
