package acme

import "testing"

func TestNormalizeCertDomains(t *testing.T) {
	domains, err := NormalizeCertDomains([]string{"example.com", "*.example.com", "nas.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 3 {
		t.Fatalf("expected 3 domains, got %d", len(domains))
	}
	if domains[1] != "*.example.com" {
		t.Fatalf("unexpected wildcard: %s", domains[1])
	}
}

func TestDomainsUnderZone(t *testing.T) {
	domains := []string{"example.com", "*.example.com", "nas.example.com"}
	if err := DomainsUnderZone(domains, "example.com"); err != nil {
		t.Fatal(err)
	}
	if err := DomainsUnderZone([]string{"other.com"}, "example.com"); err == nil {
		t.Fatal("expected zone mismatch error")
	}
}

func TestDomainsUnderZonesMultiRoot(t *testing.T) {
	zones := []string{"roven.cc", "chiak.cc"}
	if err := DomainsUnderZones([]string{"chiak.cc", "*.chiak.cc"}, zones); err != nil {
		t.Fatal(err)
	}
	if err := DomainsUnderZones([]string{"other.com"}, zones); err == nil {
		t.Fatal("expected zone mismatch error")
	}
}

func TestDecodeCertDomainsLegacy(t *testing.T) {
	got := DecodeCertDomains("", "example.com", true)
	if len(got) != 2 || got[1] != "*.example.com" {
		t.Fatalf("unexpected legacy domains: %v", got)
	}
}
