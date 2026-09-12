package logstore

import "testing"

func TestParseRealNginxAccessLine(t *testing.T) {
	line := "2026-09-12T19:55:18+08:00 127.0.0.1 GET / 200 0.000 127.0.0.1 [::1]:5173"
	entry, ok := parseAccess(line)
	if !ok {
		t.Fatal("parse failed")
	}
	if entry.Status != 200 {
		t.Fatalf("status %d", entry.Status)
	}
}
