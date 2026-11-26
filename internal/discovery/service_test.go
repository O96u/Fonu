package discovery

import (
	"strings"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestExtractTitle(t *testing.T) {
	raw := "<html><head><title>qBittorrent WebUI</title></head></html>"
	got := extractTitle(strings.ToLower(raw), raw)
	if got != "qBittorrent WebUI" {
		t.Fatalf("extractTitle() = %q, want qBittorrent WebUI", got)
	}
}

func TestDisplayNamePrefersTitle(t *testing.T) {
	got := displayName("Umami", "nginx/1.22.1", 3030)
	if got != "Umami" {
		t.Fatalf("displayName() = %q, want Umami", got)
	}
}

func TestDisplayNameFallsBackToServer(t *testing.T) {
	got := displayName("", "nginx/1.22.1", 8013)
	if got != "nginx/1.22.1" {
		t.Fatalf("displayName() = %q, want nginx/1.22.1", got)
	}
}

func TestKeywordsMatch(t *testing.T) {
	if !keywordsMatch([]string{"qbittorrent"}, "qbittorrent webui") {
		t.Fatal("expected qbittorrent keyword match")
	}
}

func TestSlugPlatform(t *testing.T) {
	got := slugPlatform("PiHost - 个人图床", 6892)
	if got == "" {
		t.Fatal("expected non-empty slug")
	}
}

func TestProbeHTTPRecognizesTLSListener(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<title>Secure service</title>")) }))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	upstream, title, _, _ := New().probeHTTP(context.Background(), u.Hostname(), port)
	if upstream != srv.URL+"/" || title != "Secure service" { t.Fatalf("got %q %q", upstream, title) }
}
