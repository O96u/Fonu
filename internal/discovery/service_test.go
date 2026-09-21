package discovery

import (
	"strings"
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
