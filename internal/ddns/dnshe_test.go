package ddns

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDNSHEUpdateRecordCreatesA(t *testing.T) {
	t.Parallel()
	var createBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := r.URL.Query().Get("endpoint")
		action := r.URL.Query().Get("action")
		if r.Header.Get("X-API-Key") != "key" || r.Header.Get("X-API-Secret") != "secret" {
			http.Error(w, `{"success":false,"error":"auth"}`, http.StatusUnauthorized)
			return
		}
		switch endpoint + ":" + action {
		case "subdomains:list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"subdomains": []map[string]any{
					{"id": 7, "full_domain": "app.de5.net"},
				},
				"pagination": map[string]any{"has_more": false},
			})
		case "dns_records:list":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "records": []any{}})
		case "dns_records:create":
			_ = json.NewDecoder(r.Body).Decode(&createBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "id": 99})
		default:
			http.Error(w, `{"success":false,"error":"unknown"}`, http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	d := &DNSHE{client: srv.Client(), baseURL: srv.URL + "?m=domain_hub"}
	cred := Credentials{Provider: "dnshe", Token: "key", Secret: "secret"}
	if err := d.UpdateRecord(context.Background(), cred, "app.de5.net", "www", "A", "1.2.3.4"); err != nil {
		t.Fatal(err)
	}
	if createBody["subdomain_id"] != float64(7) || createBody["type"] != "A" || createBody["content"] != "1.2.3.4" {
		t.Fatalf("create=%v", createBody)
	}
}

func TestDNSHERecordNameHelpers(t *testing.T) {
	t.Parallel()
	if dnsheAPIRecordName("app.de5.net", "app.de5.net") != "@" {
		t.Fatal("apex")
	}
	if dnsheAPIRecordName("app.de5.net", "www.app.de5.net") != "www" {
		t.Fatal("www")
	}
	if !dnsheRecordMatches("www.app.de5.net", "app.de5.net", "www.app.de5.net") {
		t.Fatal("match full")
	}
}

func TestDNSHEAPIError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error":"rate limited"}`))
	}))
	t.Cleanup(srv.Close)

	d := &DNSHE{client: srv.Client(), baseURL: srv.URL + "?m=domain_hub"}
	_, _, err := d.listSubdomainsPage(context.Background(), Credentials{Token: "k", Secret: "s"}, 1, 1, "")
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("err=%v", err)
	}
}
