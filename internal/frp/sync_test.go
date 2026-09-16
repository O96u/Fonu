package frp

import (
	"testing"

	"github.com/fonu/fonu/internal/proxy"
)

func TestDomainsFromProxyRules(t *testing.T) {
	rules := []proxy.Rule{
		{Enabled: true, Hosts: []proxy.Host{{Hostname: "a.example.com"}}, Domain: "b.example.com"},
		{Enabled: false, Hosts: []proxy.Host{{Hostname: "skip.example.com"}}},
		{Enabled: true, Hosts: []proxy.Host{{Hostname: "a.example.com"}, {Hostname: "c.example.com"}}},
	}
	domains := DomainsFromProxyRules(rules)
	if len(domains) != 2 {
		t.Fatalf("expected 2 domains, got %v", domains)
	}
}
