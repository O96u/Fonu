package nginx

import "testing"

func TestHostCoveredByCert(t *testing.T) {
	domains := []string{"example.com", "*.example.com"}
	if !hostCoveredByCert("app.example.com", domains) {
		t.Fatal("expected wildcard to cover subdomain")
	}
	if hostCoveredByCert("s.muksi.com", domains) {
		t.Fatal("expected typo domain to be rejected")
	}
}

func TestWildcardSingleLabelOnly(t *testing.T) {
	for _, host := range []string{"example.com", "a.b.example.com", "notexample.com"} {
		if hostCoveredByCert(host, []string{"*.example.com"}) {
			t.Fatalf("wildcard must not cover %s", host)
		}
	}
}
