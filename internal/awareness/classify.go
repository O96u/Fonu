package awareness

import (
	"strings"
)

// EventKind classifies an access log row for the dashboard.
type EventKind string

const (
	KindAccess        EventKind = "access"
	KindBlocked       EventKind = "blocked"
	KindRateLimited   EventKind = "rate_limited"
	KindAuthFailed    EventKind = "auth_failed"
	KindScanSuspected EventKind = "scan_suspected"
	KindAbnormalBurst EventKind = "abnormal_burst"
)

const (
	scan404ThresholdWindow = 10
	abnormalReqWindow      = 120
	blockedBurstWindow     = 30
)

var probePathFragments = []string{
	"/.env",
	"/.git",
	"/wp-login",
	"/wp-admin",
	"/phpmyadmin",
	"/cgi-bin/",
	"/.well-known/security.txt",
	"/actuator",
	"/solr/",
	"/manager/html",
	"/xmlrpc.php",
	"/vendor/phpunit",
}

var scannerUAFragments = []string{
	"sqlmap",
	"nmap",
	"masscan",
	"zgrab",
	"acunetix",
	"nikto",
	"nessus",
	"openvas",
	"dirbuster",
	"gobuster",
	"httpx",
	"nuclei",
	"scanner",
}

func classifyEntry(status int, path, userAgent string, ipWindow *ipWindowState) (EventKind, bool) {
	pathLower := strings.ToLower(path)
	uaLower := strings.ToLower(userAgent)

	if status == 429 {
		ipWindow.blocked++
		return KindRateLimited, true
	}
	if status == 401 {
		ipWindow.blocked++
		return KindAuthFailed, true
	}
	if status == 403 {
		ipWindow.blocked++
		return KindBlocked, true
	}

	if status == 404 {
		ipWindow.notFound++
	}

	if matchesProbePath(pathLower) || matchesScannerUA(uaLower) {
		return KindScanSuspected, true
	}
	if status == 404 && ipWindow.notFound >= scan404ThresholdWindow {
		return KindScanSuspected, true
	}
	if ipWindow.total >= abnormalReqWindow {
		return KindAbnormalBurst, true
	}
	if ipWindow.blocked >= blockedBurstWindow {
		return KindAbnormalBurst, true
	}

	return KindAccess, false
}

func matchesProbePath(path string) bool {
	for _, frag := range probePathFragments {
		if strings.Contains(path, frag) {
			return true
		}
	}
	return false
}

func matchesScannerUA(ua string) bool {
	if ua == "" || ua == "-" {
		return false
	}
	for _, frag := range scannerUAFragments {
		if strings.Contains(ua, frag) {
			return true
		}
	}
	return false
}

func eventSummary(kind EventKind, status int, path string) string {
	switch kind {
	case KindBlocked:
		return "策略拦截（403）"
	case KindRateLimited:
		return "触发限速（429）"
	case KindAuthFailed:
		return "认证失败（401）"
	case KindScanSuspected:
		return "疑似扫描：" + trimPath(path)
	case KindAbnormalBurst:
		return "高频异常访问"
	default:
		if status >= 500 {
			return "服务端错误"
		}
		return "正常访问"
	}
}

func trimPath(path string) string {
	path = strings.TrimSpace(path)
	if len(path) > 48 {
		return path[:45] + "..."
	}
	return path
}

type ipWindowState struct {
	total    int
	notFound int
	blocked  int
}
