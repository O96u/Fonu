package logstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTailLinesLargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString(formatAccessLine(time.Now(), "/line"))
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, err := tailLines(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[2], "/line") {
		t.Fatalf("expected last line, got %q", lines[2])
	}
}

func TestReadAccessLinesSinceTodayOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.Local)
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	yesterday := today.Add(-24 * time.Hour)
	lines := []string{
		formatAccessLine(yesterday, "/old"),
		formatAccessLine(today, "/today1"),
		formatAccessLine(today.Add(time.Hour), "/today2"),
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	got, err := ReadAccessLinesSince(path, start, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 today lines, got %d", len(got))
	}
	if !strings.Contains(got[0], "/today1") || !strings.Contains(got[1], "/today2") {
		t.Fatalf("unexpected lines: %v", got)
	}
}
