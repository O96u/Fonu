package ddns

import (
	"context"
	"testing"
)

// 无需 DNS 服务商账号：只模拟「账户里有哪些 zone」，验证证书 DNS-01 会往哪里写 TXT。
func TestACMEZoneFixtures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		zones    map[string]bool
		fqdn     string
		wantZone string
		wantHost string
	}{
		{
			name:     "腾讯云根域名 A.cn",
			zones:    map[string]bool{"a.cn": true},
			fqdn:     "_acme-challenge.a.cn",
			wantZone: "a.cn",
			wantHost: "_acme-challenge",
		},
		{
			name:     "Cloudflare 子域 zone example.eu.org",
			zones:    map[string]bool{"example.eu.org": true},
			fqdn:     "_acme-challenge.example.eu.org",
			wantZone: "example.eu.org",
			wantHost: "_acme-challenge",
		},
		{
			name:     "通配符证书 *.example.eu.org",
			zones:    map[string]bool{"example.eu.org": true},
			fqdn:     "_acme-challenge.example.eu.org",
			wantZone: "example.eu.org",
			wantHost: "_acme-challenge",
		},
		{
			name:     "独立托管的子域 zone",
			zones:    map[string]bool{"sub.example.com": true, "example.com": true},
			fqdn:     "_acme-challenge.www.sub.example.com",
			wantZone: "sub.example.com",
			wantHost: "_acme-challenge.www",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			zone, host, err := ResolveZoneForFQDN(context.Background(), func(candidate string) (bool, error) {
				return tc.zones[candidate], nil
			}, tc.fqdn)
			if err != nil {
				t.Fatal(err)
			}
			if zone != tc.wantZone || host != tc.wantHost {
				t.Fatalf("zone=%q host=%q, want zone=%q host=%q", zone, host, tc.wantZone, tc.wantHost)
			}
		})
	}
}
