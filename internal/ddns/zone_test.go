package ddns

import (
	"context"
	"testing"
)

func TestResolveZoneForFQDNDelegatedSubdomain(t *testing.T) {
	zones := map[string]bool{
		"sub.example.com": true,
	}
	zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
		return zones[candidate], nil
	}, "sub.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if zone != "sub.example.com" || host != "@" {
		t.Fatalf("unexpected zone=%s host=%s", zone, host)
	}
}

func TestResolveZoneForFQDNPrefersLongestZone(t *testing.T) {
	zones := map[string]bool{
		"sub.example.com": true,
		"example.com":     true,
	}
	zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
		return zones[candidate], nil
	}, "www.sub.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if zone != "sub.example.com" || host != "www" {
		t.Fatalf("unexpected zone=%s host=%s", zone, host)
	}
}

func TestResolveZoneForFQDNApexACMEFQDN(t *testing.T) {
	zones := map[string]bool{
		"a.cn": true,
	}
	zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
		return zones[candidate], nil
	}, "_acme-challenge.a.cn")
	if err != nil {
		t.Fatal(err)
	}
	if zone != "a.cn" || host != "_acme-challenge" {
		t.Fatalf("unexpected zone=%s host=%s", zone, host)
	}
}

func TestResolveZoneForFQDNDelegatedACMEFQDN(t *testing.T) {
	zones := map[string]bool{
		"example.eu.org": true,
	}
	zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
		return zones[candidate], nil
	}, "_acme-challenge.example.eu.org")
	if err != nil {
		t.Fatal(err)
	}
	if zone != "example.eu.org" || host != "_acme-challenge" {
		t.Fatalf("unexpected zone=%s host=%s", zone, host)
	}
}

func TestResolveZoneForFQDNClassicSubdomain(t *testing.T) {
	zones := map[string]bool{
		"example.com": true,
	}
	zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
		return zones[candidate], nil
	}, "s.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if zone != "example.com" || host != "s" {
		t.Fatalf("unexpected zone=%s host=%s", zone, host)
	}
}

func TestResolveZoneForFQDNWildcard(t *testing.T) {
	zones := map[string]bool{
		"sub.example.com": true,
	}
	zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
		return zones[candidate], nil
	}, "*.sub.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if zone != "sub.example.com" || host != "*" {
		t.Fatalf("unexpected zone=%s host=%s", zone, host)
	}
}

func TestFQDNFromRecordStoredFQDN(t *testing.T) {
	cfg := Config{RootDomain: "example.com"}
	if FQDNFromRecord(cfg, "sub.example.com") != "sub.example.com" {
		t.Fatal("expected stored fqdn preserved")
	}
}
