package logstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHourlyAccessCountsCalendarDay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	now := time.Now()
	todayMorning := time.Date(now.Year(), now.Month(), now.Day(), 8, 15, 0, 0, now.Location())
	todayNoon := time.Date(now.Year(), now.Month(), now.Day(), 11, 30, 0, 0, now.Location())
	yesterday := todayMorning.Add(-24 * time.Hour)
	lines := []string{
		formatAccessLine(todayMorning),
		formatAccessLine(todayMorning.Add(30 * time.Minute)),
		formatAccessLine(todayNoon),
		formatAccessLine(yesterday),
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	got, labels := HourlyAccessCounts(path)
	if len(got) != 12 || len(labels) != 12 {
		t.Fatalf("expected 12 buckets, got %d counts / %d labels", len(got), len(labels))
	}
	if labels[0] != "00:00" || labels[11] != "22:00" {
		t.Fatalf("unexpected labels: %v", labels)
	}
	sum := 0
	for _, n := range got {
		sum += n
	}
	if sum != 3 {
		t.Fatalf("expected 3 requests today, got %d (%v)", sum, got)
	}
	currentBucket := now.Hour() / 2
	for i := currentBucket + 1; i < 12; i++ {
		if got[i] != 0 {
			t.Fatalf("future bucket %d should be 0, got %d", i, got[i])
		}
	}
}

func formatAccessLine(at time.Time) string {
	return at.Format(time.RFC3339) + " app.example.com GET / 200 0.010 1.2.3.4 127.0.0.1:8080"
}
