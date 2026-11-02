package iplocate

import (
	"strings"
	"unicode"
)

func formatForeignLabel(country, province, city, isp string) string {
	c := translateCountry(country)
	if c == "" {
		return localizeISP(isp)
	}
	return c
}

func localizeISP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || isCountryCode(raw) {
		return ""
	}
	if containsHan(raw) {
		return raw
	}
	key := normKey(raw)
	if v, ok := ispNames[key]; ok {
		return v
	}
	for k, v := range ispNames {
		if strings.Contains(key, k) {
			return v
		}
	}
	return raw
}

func localizeLabel(raw string, table map[string]string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0" {
		return ""
	}
	if containsHan(raw) {
		return raw
	}
	if v, ok := table[normKey(raw)]; ok {
		return v
	}
	return ""
}

func normKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func equalFold(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

var regionNames = map[string]string{
	"california":  "加利福尼亚",
	"virginia":    "弗吉尼亚",
	"oregon":      "俄勒冈",
	"washington":  "华盛顿州",
	"texas":       "德克萨斯",
	"new york":    "纽约州",
	"florida":     "佛罗里达",
	"illinois":    "伊利诺伊",
	"georgia":     "佐治亚",
	"ohio":        "俄亥俄",
	"colorado":    "科罗拉多",
	"arizona":     "亚利桑那",
	"nevada":      "内华达",
	"massachusetts": "马萨诸塞",
	"pennsylvania": "宾夕法尼亚",
	"michigan":    "密歇根",
	"new jersey":  "新泽西",
	"north carolina": "北卡罗来纳",
	"seoul":       "首尔",
	"tokyo":       "东京",
	"osaka":       "大阪",
}

var cityNames = map[string]string{
	"san jose":      "圣何塞",
	"san francisco": "旧金山",
	"los angeles":   "洛杉矶",
	"seattle":       "西雅图",
	"portland":      "波特兰",
	"ashburn":       "阿什本",
	"chicago":       "芝加哥",
	"dallas":        "达拉斯",
	"houston":       "休斯顿",
	"miami":         "迈阿密",
	"new york":      "纽约",
	"silicon valley": "硅谷",
	"frankfurt":     "法兰克福",
	"amsterdam":     "阿姆斯特丹",
	"london":        "伦敦",
	"paris":         "巴黎",
	"singapore":     "新加坡",
	"hong kong":     "香港",
	"taipei":        "台北",
	"tokyo":         "东京",
	"seoul":         "首尔",
	"sydney":        "悉尼",
	"mumbai":        "孟买",
	"bangalore":     "班加罗尔",
}

var ispNames = map[string]string{
	"amazon":        "亚马逊",
	"aws":           "亚马逊 AWS",
	"google":        "谷歌",
	"microsoft":     "微软",
	"azure":         "微软 Azure",
	"cloudflare":    "Cloudflare",
	"digitalocean":  "DigitalOcean",
	"linode":        "Linode",
	"akamai":        "Akamai",
	"oracle":        "甲骨文",
	"alibaba":       "阿里云",
	"tencent":       "腾讯云",
	"huawei":        "华为云",
	"baidu":         "百度云",
	"fastly":        "Fastly",
	"hetzner":       "Hetzner",
	"ovh":           "OVH",
	"choopa":        "Vultr",
	"vultr":         "Vultr",
	"hostinger":     "Hostinger",
	"godaddy":       "GoDaddy",
	"apple":         "苹果",
	"facebook":      "Meta",
	"meta":          "Meta",
}
