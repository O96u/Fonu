package frp

import (
	"strings"

	"github.com/fonu/fonu/internal/proxy"
)

func DomainsFromProxyRules(rules []proxy.Rule) []string {
	seen := make(map[string]bool)
	var domains []string
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		for _, host := range rule.Hostnames() {
			if host == "" || seen[host] {
				continue
			}
			seen[host] = true
			domains = append(domains, host)
		}
	}
	return domains
}

func MergeDomains(existing, incoming []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, list := range [][]string{existing, incoming} {
		for _, d := range list {
			d = strings.TrimSpace(d)
			key := strings.ToLower(d)
			if d == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, d)
		}
	}
	return out
}
