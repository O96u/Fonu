package logstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHourlyAccessCounts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	today := time.Now().Format("2006-01-02")
	lines := []string{
		today + "T08:15:00+08:00 app.example.com GET / 200 0.010 1.2.3.4 127.0.0.1:8080",
		today + "T08:45:00+08:00 app.example.com GET /api 200 0.010 1.2.3.4 127.0.0.1:8080",
		today + "T18:30:00+08:00 app.example.com GET / 200 0.010 1.2.3.4 127.0.0.1:8080",
		"2020-01-01T18:30:00+08:00 app.example.com GET / 200 0.010 1.2.3.4 127.0.0.1:8080",
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	got := HourlyAccessCounts(path)
	if got[4] != 2 {
		t.Fatalf("bucket 08-10 expected 2, got %d (%v)", got[4], got)
	}
	if got[9] != 1 {
		t.Fatalf("bucket 18-20 expected 1, got %d (%v)", got[9], got)
	}
}
