package ddns

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type dnsPodEmptyRecordTransport func(*http.Request) (*http.Response, error)

func (f dnsPodEmptyRecordTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDNSPodEmptyRecordsCanCreate(t *testing.T) {
	for _, tc := range []struct {
		name, listMessage, operation, recordType string
		wantCreate, wantError bool
	}{
		{"first DDNS A", "记录列表为空", "ddns", "A", true, false},
		{"first DDNS AAAA", "记录列表为空", "ddns", "AAAA", true, false},
		{"first ACME TXT", "记录列表为空", "acme", "TXT", true, false},
		{"permission error", "没有权限", "ddns", "A", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			created := false
			d := &DNSPod{client: &http.Client{Transport: dnsPodEmptyRecordTransport(func(req *http.Request) (*http.Response, error) {
				body := ""
				switch {
				case strings.HasSuffix(req.URL.Path, "Domain.List"):
					body = `{"status":{"code":"1"},"domains":[{"id":123,"name":"mydomain.eu.org"}]}`
				case strings.HasSuffix(req.URL.Path, "Record.List"):
					body = `{"status":{"code":"10","message":"` + tc.listMessage + `"}}`
				case strings.HasSuffix(req.URL.Path, "Record.Create"):
					created = true
					if err := req.ParseForm(); err != nil {
						t.Fatal(err)
					}
					if req.Form.Get("domain_id") != "123" || req.Form.Get("sub_domain") != "h" && req.Form.Get("sub_domain") != "_acme-challenge.h" || req.Form.Get("record_type") != tc.recordType {
						t.Errorf("unexpected record creation: %v", req.Form)
					}
					body = `{"status":{"code":"1"},"record":{"id":"456"}}`
				default:
					t.Fatalf("unexpected API path: %s", req.URL.Path)
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})}}
			ctx := context.Background()
			cred := Credentials{TokenID:"test", Token:"test"}
			var err error
			if tc.operation == "acme" {
				err = d.UpsertTXTRecord(ctx, cred, "_acme-challenge.h.mydomain.eu.org", "token")
			} else {
				zone, host, lookupErr := ResolveZoneForFQDN(ctx, func(candidate string)(bool,error){return d.HasZone(ctx,cred,candidate)}, "h.mydomain.eu.org")
				if lookupErr != nil { t.Fatal(lookupErr) }
				var current string
				current, err = d.GetRecordIP(ctx, cred, zone, host, tc.recordType)
				if err == nil {
					if current != "" { t.Fatalf("unexpected current record %s", current) }
					err = d.UpdateRecord(ctx, cred, zone, host, tc.recordType, "test-address")
				}
			}
			if (err != nil) != tc.wantError || created != tc.wantCreate {
				t.Fatalf("created=%t error=%v", created, err)
			}
		})
	}
}
