package ddns

import "testing"

func TestManagedDNSZonesMultiRoot(t *testing.T) {
	cfg := Config{
		RootDomain:  "roven.cc",
		RecordNames: []string{"roven.cc", "chiak.cc"},
	}
	zones := cfg.ManagedDNSZones()
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %v", zones)
	}
	if !cfg.CoversDomain("chiak.cc") || !cfg.CoversDomain("*.chiak.cc") {
		t.Fatal("expected chiak.cc coverage")
	}
	if cfg.CoversDomain("other.com") {
		t.Fatal("unexpected coverage")
	}
}
