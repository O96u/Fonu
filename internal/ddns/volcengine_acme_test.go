package ddns

import (
	"context"
	"testing"
)

func TestResolveZoneForACMEChallenge(t *testing.T) {
	zones := map[string]bool{
		"example.com": true,
	}
	zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
		return zones[candidate], nil
	}, "_acme-challenge.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if zone != "example.com" || host != "_acme-challenge" {
		t.Fatalf("unexpected zone=%s host=%s", zone, host)
	}
}
