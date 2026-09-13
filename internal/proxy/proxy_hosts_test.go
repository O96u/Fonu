package proxy

import (
	"testing"
)

func TestParseHostsAllowsSameHostnameDifferentPorts(t *testing.T) {
	hosts, err := parseHosts([]string{"example.com", "example.com:6893"}, 8011)
	if err != nil {
		t.Fatalf("parseHosts failed: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
	if hostEffectivePort(hosts[0], 8011) != 8011 {
		t.Fatalf("expected default port 8011, got %d", hostEffectivePort(hosts[0], 8011))
	}
	if hostEffectivePort(hosts[1], 8011) != 6893 {
		t.Fatalf("expected custom port 6893, got %d", hostEffectivePort(hosts[1], 8011))
	}
}

func TestParseHostsRejectsDuplicateBinding(t *testing.T) {
	_, err := parseHosts([]string{"example.com", "example.com"}, 8011)
	if err == nil {
		t.Fatal("expected duplicate binding error")
	}
}
