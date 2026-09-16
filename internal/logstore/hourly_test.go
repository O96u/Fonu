package logstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHourlyAccessCountsRolling24h(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	now := time.Now()
	lines := []string{
		formatAccessLine(now.Add(-20*time.Hour)),
		formatAccessLine(now.Add(-19*time.Hour)),
		formatAccessLine(now.Add(-2*time.Hour)),
		formatAccessLine(now.Add(-30*time.Minute)),
		formatAccessLine(now.Add(-30*time.Hour)), // outside rolling window
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	got, labels := HourlyAccessCounts(path)
	if len(got) != 12 || len(labels) != 12 {
		t.Fatalf("expected 12 buckets, got %d counts / %d labels", len(got), len(labels))
	}
	sum := 0
	for _, n := range got {
		sum += n
	}
	if sum != 4 {
		t.Fatalf("expected 4 requests in window, got %d (%v)", sum, got)
	}
	latest := got[11]
	if latest < 1 {
		t.Fatalf("latest bucket expected recent traffic, got %d (%v)", latest, got)
	}
}

func formatAccessLine(at time.Time) string {
	return at.Format(time.RFC3339) + " app.example.com GET / 200 0.010 1.2.3.4 127.0.0.1:8080"
}
