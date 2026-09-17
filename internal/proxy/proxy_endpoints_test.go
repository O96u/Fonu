package proxy

import "testing"

func TestRuleEndpointsUsesListenPort(t *testing.T) {
	rule := Rule{
		ListenPort: 6893,
		Hosts: []Host{
			{Hostname: "xxx.com"},
			{Hostname: "xxx.com", ListenPort: intPtr(8081)},
		},
	}
	endpoints := rule.Endpoints()
	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}
	if endpoints[0].Hostname != "xxx.com" || endpoints[0].Port != 6893 {
		t.Fatalf("unexpected first endpoint: %+v", endpoints[0])
	}
	if endpoints[1].Port != 8081 {
		t.Fatalf("expected custom port 8081, got %d", endpoints[1].Port)
	}
}

func intPtr(v int) *int { return &v }
