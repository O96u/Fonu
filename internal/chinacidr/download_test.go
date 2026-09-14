package chinacidr

import (
	"context"
	"testing"
)

func TestDownloadDefaultV6Sources(t *testing.T) {
	ctx := context.Background()
	lines, err := downloadFromSources(ctx, defaultV6URLs...)
	if err != nil {
		t.Fatalf("download v6: %v", err)
	}
	if len(lines) < 10 {
		t.Fatalf("expected at least 10 v6 entries, got %d", len(lines))
	}
}

func TestGhfastProxyURL(t *testing.T) {
	raw := "https://raw.githubusercontent.com/foo/bar.txt"
	got := ghfastProxyURL(raw)
	want := ghfastProxyPrefix + raw
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if ghfastProxyURL(want) != "" {
		t.Fatal("expected no double proxy")
	}
}
