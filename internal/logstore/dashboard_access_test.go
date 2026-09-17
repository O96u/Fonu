package logstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDashboardAccessFilterSkipsNoisePaths(t *testing.T) {
	if isDashboardAccess(AccessEntry{Path: "/login"}) != true {
		t.Fatal("expected /login to count")
	}
	if isDashboardAccess(AccessEntry{Path: "/favicon.ico"}) {
		t.Fatal("expected favicon to be excluded")
	}
	if isDashboardAccess(AccessEntry{Path: "/fonu-errors/error.png"}) {
		t.Fatal("expected error asset to be excluded")
	}
}

func TestCountAccessForDayExcludesNoisePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	now := time.Now()
	line := formatAccessLine(now, "/login")
	noise := formatAccessLine(now, "/favicon.ico")
	if err := os.WriteFile(path, []byte(strings.Join([]string{line, noise}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	total, _, _ := CountAccessForDay(path, now)
	if total != 1 {
		t.Fatalf("expected 1 dashboard request, got %d", total)
	}
}
