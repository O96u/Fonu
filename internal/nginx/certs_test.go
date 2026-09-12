package nginx

import "testing"

func TestHostCoveredByCert(t *testing.T) {
	domains := []string{"muxui.com", "*.muxui.com"}
	if !hostCoveredByCert("s.muxui.com", domains) {
		t.Fatal("expected wildcard to cover subdomain")
	}
	if hostCoveredByCert("s.muksi.com", domains) {
		t.Fatal("expected typo domain to be rejected")
	}
}
