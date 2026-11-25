package ddns

import (
 "encoding/json"
 "testing"
)

func TestDNSHEFQDN(t *testing.T) {
	cases := []struct {
		root, record, want string
	}{
		{"example.cc.cd", "@", "example.cc.cd"},
		{"example.cc.cd", "", "example.cc.cd"},
		{"example.cc.cd", "nas", "nas.example.cc.cd"},
		{"example.cc.cd", "a.b", "a.b.example.cc.cd"},
		{"example.cc.cd", "nas.example.cc.cd", "nas.example.cc.cd"},
		{"example.cc.cd", "nas.example.cc.cd.", "nas.example.cc.cd"},
	}
	for _, c := range cases {
		if got := dnsheFQDN(c.root, c.record); got != c.want {
			t.Errorf("dnsheFQDN(%q, %q) = %q, want %q", c.root, c.record, got, c.want)
		}
	}
}

func TestDNSHERecordNameParam(t *testing.T) {
	if got := dnsheRecordNameParam("example.cc.cd", "example.cc.cd"); got != "" {
		t.Errorf("root record name param = %q, want empty", got)
	}
	if got := dnsheRecordNameParam("example.cc.cd", "_acme-challenge.example.cc.cd"); got != "_acme-challenge" {
		t.Errorf("txt record name param = %q, want relative name", got)
	}
	if got := dnsheRecordNameParam("example.cc.cd", "nas.example.cc.cd"); got != "nas" {
		t.Errorf("sub record name param = %q, want relative name", got)
	}
}

func TestDNSHEGiftResponse(t *testing.T) {
	for _, raw := range []string{
		`{"success":true,"code":"test-code","full_domain":"example.cc.cd"}`,
		`{"success":true,"data":{"code":"test-code","full_domain":"example.cc.cd"}}`,
		`{"success":true,"gift":{"code":"test-code","full_domain":"example.cc.cd"}}`,
		`{"success":true,"data":{"gift":{"code":"test-code","full_domain":"example.cc.cd"}}}`,
	} {
		var response dnsheGiftResponse
		if err := json.Unmarshal([]byte(raw), &response); err != nil { t.Fatal(err) }
		gift := response.result()
		if gift.Code != "test-code" || gift.FullDomain != "example.cc.cd" { t.Fatal("gift fields missing") }
	}
}
