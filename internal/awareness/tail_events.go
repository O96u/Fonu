package awareness

import (
	"strings"
	"time"

	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/proxy"
)

// TailRecentEvents returns the newest dashboard access rows from the end of access.log.
// No time-range or risk/intercept filter — only skips non-dashboard noise (favicon, etc.).
func TailRecentEvents(path string, bindings proxy.BindingIndex, limit int) []Event {
	if limit <= 0 {
		limit = maxRecentEvents
	}
	maxScan := limit * 50
	if maxScan < 800 {
		maxScan = 800
	}
	if maxScan > 8000 {
		maxScan = 8000
	}

	lines, err := logstore.TailLines(path, maxScan)
	if err != nil {
		return nil
	}

	out := make([]Event, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		entry, ok := logstore.ParseAccess(lines[i])
		if !ok || !logstore.IsDashboardAccess(entry) {
			continue
		}
		at, ok := parseAccessTime(entry.Time)
		if !ok {
			at = time.Now().UTC()
		}
		service := proxy.ResolveServiceLabel(entry.Domain, entry.ServerPort, entry.Upstream, bindings)
		ip := strings.TrimSpace(entry.ClientIP)
		if ip == "" {
			ip = "—"
		}
		kind, summary := ClassifyAccess(entry)
		out = append(out, Event{
			At:       at.UTC().Format(time.RFC3339),
			ClientIP: ip,
			Service:  service,
			Domain:   entry.Domain,
			Path:     entry.Path,
			Status:   entry.Status,
			Kind:     kind,
			Summary:  summary,
		})
	}
	return out
}
