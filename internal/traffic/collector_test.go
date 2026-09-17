package traffic

import (
	"testing"
	"time"

	"github.com/fonu/fonu/internal/proxy"
)

func TestCollectorAttributesConnectionsByHostAndPort(t *testing.T) {
	c := NewCollector(nil, "")
	now := time.Now()
	line6893 := "2026-09-17T08:30:00+08:00 xxx.com 6893 GET /login 200 0.010 218.88.23.20 192.168.8.3:6893 100 200"
	line8443 := "2026-09-17T08:30:01+08:00 xxx.com 8443 GET / 200 0.010 218.88.23.20 192.168.8.3:8443 100 200"

	c.ingestLine(line6893, true)
	c.ingestLine(line8443, true)

	rules := []proxy.Rule{
		{
			ID:         1,
			ListenPort: 6893,
			Hosts:      []proxy.Host{{Hostname: "xxx.com"}},
		},
		{
			ID:         2,
			ListenPort: 8443,
			Hosts:      []proxy.Host{{Hostname: "xxx.com"}},
		},
	}

	stats := c.SnapshotForRules(rules)
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}
	byID := map[int64]RuleTraffic{}
	for _, s := range stats {
		byID[s.RuleID] = s
	}
	if byID[1].Connections != 1 {
		t.Fatalf("rule 6893 connections=%d want 1", byID[1].Connections)
	}
	if byID[2].Connections != 1 {
		t.Fatalf("rule 8443 connections=%d want 1", byID[2].Connections)
	}
	_ = now
}

func TestEndpointKeyIncludesPort(t *testing.T) {
	if got := EndpointKey("XXX.com", 6893); got != "xxx.com:6893" {
		t.Fatalf("got %q", got)
	}
}
