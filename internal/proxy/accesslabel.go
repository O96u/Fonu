package proxy

import (
	"fmt"
	"strings"
)

// BindingIndex maps host:port to a display name for access log rows.
type BindingIndex map[string]string

func bindingKey(host string, port int) string {
	return fmt.Sprintf("%s:%d", normalizeAccessHost(host), port)
}

func normalizeAccessHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.Index(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}

// BuildBindingIndex indexes enabled proxy rules by hostname and listen port.
func BuildBindingIndex(rules []Rule) BindingIndex {
	index := make(BindingIndex)
	for _, rule := range rules {
		fallback := strings.TrimSpace(rule.Domain)
		if fallback == "" {
			fallback = strings.TrimSpace(rule.Upstream)
		}
		if fallback == "" {
			fallback = fmt.Sprintf("规则 #%d", rule.ID)
		}
		name := strings.TrimSpace(rule.Name)
		if name == "" {
			name = fallback
		}
		hosts := rule.Hosts
		if len(hosts) == 0 && rule.Domain != "" {
			hosts = []Host{{Hostname: rule.Domain}}
		}
		for _, host := range hosts {
			hostname := strings.TrimSpace(host.Hostname)
			if hostname == "" {
				continue
			}
			port := rule.ListenPort
			if host.ListenPort != nil && *host.ListenPort > 0 {
				port = *host.ListenPort
			}
			if port <= 0 {
				continue
			}
			index[bindingKey(hostname, port)] = name
		}
	}
	return index
}

// ResolveServiceLabel maps an access log row to a human-readable service name.
func ResolveServiceLabel(domain string, serverPort int, upstream string, index BindingIndex) string {
	host := normalizeAccessHost(domain)
	if host != "" && serverPort > 0 {
		if name, ok := index[bindingKey(host, serverPort)]; ok {
			return name
		}
	}
	if host != "" {
		var matches []string
		for key, name := range index {
			if strings.HasPrefix(key, host+":") {
				matches = append(matches, name)
			}
		}
		if len(matches) == 1 {
			return matches[0]
		}
	}
	upstream = strings.TrimSpace(upstream)
	if upstream != "" && upstream != "-" {
		return strings.TrimPrefix(strings.TrimPrefix(upstream, "http://"), "https://")
	}
	if host != "" && serverPort > 0 {
		return fmt.Sprintf("%s:%d", host, serverPort)
	}
	if strings.TrimSpace(domain) != "" {
		return strings.TrimSpace(domain)
	}
	return "—"
}
