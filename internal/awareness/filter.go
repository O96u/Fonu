package awareness

import (
	"strings"

	"github.com/fonu/fonu/internal/logstore"
)

// ClassifyAccess labels a single access row (no multi-request burst context).
func ClassifyAccess(entry logstore.AccessEntry) (EventKind, string) {
	ipWin := &ipWindowState{}
	kind, _ := classifyEntry(entry.Status, entry.Path, entry.UserAgent, ipWin)
	return kind, eventSummary(kind, entry.Status, entry.Path)
}

// MatchKindFilter returns true if kind matches the API/UI filter key.
func MatchKindFilter(kind EventKind, filter string) bool {
	filter = strings.TrimSpace(strings.ToLower(filter))
	switch filter {
	case "", "all":
		return true
	case "intercept", "blocked", "block":
		return kind == KindBlocked || kind == KindRateLimited || kind == KindAuthFailed
	case "403", "deny":
		return kind == KindBlocked
	case "429", "rate_limited":
		return kind == KindRateLimited
	case "401", "auth_failed":
		return kind == KindAuthFailed
	case "scan", "scan_suspected":
		return kind == KindScanSuspected
	case "abnormal", "abnormal_burst":
		return kind == KindAbnormalBurst
	case "risk":
		return kind != KindAccess
	default:
		return string(kind) == filter
	}
}
