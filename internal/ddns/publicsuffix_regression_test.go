package ddns

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestDDNSPublicSuffixAndLegacyFQDN(t *testing.T) {
	for _, tc := range []struct{ domain, root string }{
		{"h.mydomain.eu.org", "mydomain.eu.org"},
		{"nas.example.co.uk", "example.co.uk"},
		{"nas.example.com.cn", "example.com.cn"},
		{"api.nas.example.com", "example.com"},
		{"*.mydomain.eu.org", "mydomain.eu.org"},
	} {
		root, names, err := ParseDomainLines([]string{tc.domain})
		if err != nil || root != tc.root || len(names) != 1 || names[0] != tc.domain {
			t.Fatalf("%s: root=%q names=%v err=%v", tc.domain, root, names, err)
		}
	}
	legacy := Config{RootDomain: "eu.org", RecordNames: []string{"h.mydomain.eu.org", "other.example.co.uk"}}
	if got := legacy.DomainLines(); len(got) != 2 || got[0] != "h.mydomain.eu.org" || got[1] != "other.example.co.uk" {
		t.Fatalf("legacy FQDNs changed: %v", got)
	}
	if legacy.CoversDomain("unrelated.eu.org") || !legacy.CoversDomain("mydomain.eu.org") {
		t.Fatal("legacy public suffix must not cover unrelated tenants")
	}
}

func TestResolvedRelativeDNSName(t *testing.T) {
	for _, host := range []string{"api.nas", "a.b.c", "_acme-challenge.api", "*.nas", "@"} {
		want := host + ".example.com"
		if host == "@" {
			want = "example.com"
		}
		if got := fqdnForRecord("example.com", host); got != want {
			t.Errorf("host %s: got %q want %q", host, got, want)
		}
	}
	zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
		return candidate == "mydomain.eu.org", nil
	}, "api.nas.mydomain.eu.org")
	if err != nil || zone != "mydomain.eu.org" || host != "api.nas" || fqdnForRecord(zone, host) != "api.nas.mydomain.eu.org" {
		t.Fatalf("zone=%q host=%q err=%v", zone, host, err)
	}
}

func TestDNSPodAccountErrorRemainsFatal(t *testing.T) {
	raw := errors.New("could not get domains: 记录列表为空")
	err := dnsPodDomainListError(raw)
	if !errors.Is(err, raw) || !strings.Contains(err.Error(), "账号权限") {
		t.Fatalf("missing DNSPod diagnostic: %v", err)
	}
	calls := 0
	_, _, got := ResolveZoneForFQDN(context.Background(), func(string) (bool, error) {
		calls++
		return false, err
	}, "h.mydomain.eu.org")
	if !errors.Is(got, raw) || calls != 1 {
		t.Fatalf("account error must stop lookup, calls=%d err=%v", calls, got)
	}
}
