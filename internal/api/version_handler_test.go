package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fonu/fonu/internal/config"
	"github.com/fonu/fonu/internal/version"
)

func TestVersionEndpoint(t *testing.T) {
	version.Version = "0.1.6"
	cfg := config.Config{NginxDefaultHTTPPort: 18080, NginxDefaultHTTPSPort: 9443}
	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()
	Version(cfg)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["version"] != "0.1.6" {
		t.Fatalf("version %q", body["version"])
	}
	if body["nginx_http_port"] != float64(18080) {
		t.Fatalf("nginx_http_port %v", body["nginx_http_port"])
	}
	if body["nginx_https_port"] != float64(9443) {
		t.Fatalf("nginx_https_port %v", body["nginx_https_port"])
	}
}
