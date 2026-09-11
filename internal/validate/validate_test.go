package validate

import "testing"

func TestDomain(t *testing.T) {
	if err := Domain("nas.example.com"); err != nil {
		t.Fatalf("expected valid domain: %v", err)
	}
	if err := Domain("bad..domain"); err == nil {
		t.Fatal("expected invalid domain")
	}
}

func TestUpstream(t *testing.T) {
	normalized, err := Upstream("http://192.168.1.10:5666")
	if err != nil {
		t.Fatalf("expected valid upstream: %v", err)
	}
	if normalized != "http://192.168.1.10:5666" {
		t.Fatalf("unexpected normalized upstream: %s", normalized)
	}
	if _, err := Upstream("ftp://example.com"); err == nil {
		t.Fatal("expected scheme validation error")
	}
	if _, err := Upstream("http://example.com/evil;rm -rf /"); err == nil {
		t.Fatal("expected path rejection")
	}
}
