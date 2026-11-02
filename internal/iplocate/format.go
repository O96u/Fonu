package iplocate

import (
	"net"
	"strings"
)

// FormatRegion turns ip2region "国家|区域|省份|城市|ISP" into a short UI label.
// Domestic: 广东省 · 深圳市 · 电信（直辖市只显示一次市名）.
func FormatRegion(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return ""
	}
	parts := strings.Split(raw, "|")
	for len(parts) < 5 {
		parts = append(parts, "")
	}
	country := trimField(parts[0])
	region := trimField(parts[1])
	province := trimField(parts[2])
	city := trimField(parts[3])
	isp := trimField(parts[4])
	if province == "" && region != "" {
		province = region
	}

	// Some v4 rows put ISP in the city slot and country code in ISP slot.
	if country == "中国" && isCNISPName(city) && (isp == "" || isCountryCode(isp)) {
		isp = city
		city = ""
	}
	if isCNISPName(province) && city == "" && isp == "" {
		isp = province
		province = ""
	}

	if country == "" {
		return ""
	}
	if country == "中国" {
		return formatChinaLabel(province, city, isp)
	}

	return formatForeignLabel(country, province, city, isp)
}

func formatChinaLabel(province, city, isp string) string {
	segments := make([]string, 0, 3)
	if province != "" {
		segments = append(segments, normalizeCNPlace(province))
	}
	if city != "" && !samePlace(province, city) && !isCNISPName(city) {
		segments = append(segments, normalizeCNPlace(city))
	}
	label := strings.Join(segments, " · ")
	if isp != "" && !isCountryCode(isp) {
		if label == "" {
			return isp
		}
		return label + " · " + isp
	}
	return label
}

func trimField(s string) string {
	s = strings.TrimSpace(s)
	if s == "0" {
		return ""
	}
	return s
}

func normalizeCNPlace(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return s
}

func samePlace(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == b {
		return true
	}
	aCore := strings.TrimSuffix(strings.TrimSuffix(a, "省"), "市")
	bCore := strings.TrimSuffix(strings.TrimSuffix(b, "省"), "市")
	return aCore != "" && aCore == bCore
}

func isCNISPName(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, kw := range []string{"电信", "联通", "移动", "铁通", "广电", "鹏博士", "教育网", "长城宽带", "广电网"} {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func isCountryCode(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) < 2 || len(s) > 3 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// LabelForPrivate returns a label for non-public addresses.
func LabelForPrivate(ip string) (string, bool) {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return "", false
	}
	if parsed.IsLoopback() {
		return "本机", true
	}
	if parsed.IsPrivate() || parsed.IsLinkLocalUnicast() {
		return "内网", true
	}
	return "", false
}
