package awareness

import (
	"strings"

	"github.com/fonu/fonu/internal/logstore"
	"github.com/fonu/fonu/internal/proxy"
)

// AccessLogFilter selects rows from recent access.log lines.
type AccessLogFilter struct {
	Limit     int
	Keyword   string
	Status    int
	ClientIP  string
	Kind      string
	MaxScan   int
}

// EnrichedAccessLog is one dashboard access row with service label and block reason.
type EnrichedAccessLog struct {
	logstore.AccessEntry
	Service       string `json:"service,omitempty"`
	AwarenessKind string `json:"awareness_kind,omitempty"`
	BlockReason   string `json:"block_reason,omitempty"`
	Geo           string `json:"geo,omitempty"`
}

// FilterAccessLog reads recent lines from the end of the file and returns matching rows.
func FilterAccessLog(path string, filter AccessLogFilter, bindings proxy.BindingIndex) ([]EnrichedAccessLog, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	maxScan := filter.MaxScan
	if maxScan <= 0 {
		maxScan = limit * 40
	}
	if maxScan < 500 {
		maxScan = 500
	}
	if maxScan > 8000 {
		maxScan = 8000
	}

	lines, err := logstore.TailLines(path, maxScan)
	if err != nil {
		return nil, err
	}

	kw := strings.ToLower(strings.TrimSpace(filter.Keyword))
	ipNeedle := strings.TrimSpace(filter.ClientIP)
	kindFilter := strings.TrimSpace(filter.Kind)

	out := make([]EnrichedAccessLog, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(out) < limit; i-- {
		entry, ok := logstore.ParseAccess(lines[i])
		if !ok || !logstore.IsDashboardAccess(entry) {
			continue
		}
		if filter.Status > 0 && entry.Status != filter.Status {
			continue
		}
		if ipNeedle != "" && !strings.EqualFold(strings.TrimSpace(entry.ClientIP), ipNeedle) {
			continue
		}
		service := proxy.ResolveServiceLabel(entry.Domain, entry.ServerPort, entry.Upstream, bindings)
		kind, reason := ClassifyAccess(entry)
		if kindFilter != "" && !MatchKindFilter(kind, kindFilter) {
			continue
		}
		if kw != "" {
			hay := strings.ToLower(strings.Join([]string{
				service,
				entry.Domain,
				entry.Path,
				entry.ClientIP,
				entry.Method,
				reason,
				string(kind),
			}, " "))
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		row := EnrichedAccessLog{
			AccessEntry:   entry,
			Service:       service,
			AwarenessKind: string(kind),
		}
		if kind != KindAccess {
			row.BlockReason = reason
		}
		out = append(out, row)
	}
	return out, nil
}
