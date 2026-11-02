package analytics

import (
	"sort"
	"strings"
)

func classifyUA(ua string) string {
	l := strings.ToLower(ua)
	switch {
	case strings.Contains(l, "android"):
		return "Android"
	case strings.Contains(l, "iphone") || strings.Contains(l, "ipad") || strings.Contains(l, "ios"):
		return "iOS"
	case strings.Contains(l, "edg/"):
		return "Edge"
	case strings.Contains(l, "chrome/") && !strings.Contains(l, "edg/"):
		return "Chrome"
	case strings.Contains(l, "firefox/"):
		return "Firefox"
	case strings.Contains(l, "safari/") && !strings.Contains(l, "chrome/"):
		return "Safari"
	case strings.Contains(l, "windows"):
		return "Windows"
	case strings.Contains(l, "mac os"):
		return "macOS"
	case ua == "" || ua == "-":
		return "未知"
	default:
		return "其他"
	}
}

func sortClientRows(rows []ClientRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Requests == rows[j].Requests {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Requests > rows[j].Requests
	})
}
