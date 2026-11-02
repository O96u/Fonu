package awareness

import "strings"

// ClassifyStatic assigns kind for persisted rows (no sliding IP window).
func ClassifyStatic(status int, path, userAgent string) EventKind {
	pathLower := strings.ToLower(path)
	uaLower := strings.ToLower(userAgent)

	switch status {
	case 429:
		return KindRateLimited
	case 401:
		return KindAuthFailed
	case 403:
		return KindBlocked
	}
	if matchesProbePath(pathLower) || matchesScannerUA(uaLower) {
		return KindScanSuspected
	}
	return KindAccess
}

// IsRiskKind reports whether kind counts as malicious/risk KPI.
func IsRiskKind(kind EventKind) bool {
	switch kind {
	case KindBlocked, KindRateLimited, KindAuthFailed, KindScanSuspected, KindAbnormalBurst:
		return true
	default:
		return false
	}
}
