package awareness

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTailRecentEventsAllStatuses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	loc := time.FixedZone("CST", 8*3600)
	now := time.Now().In(loc)
	t1 := now.Add(-3 * time.Hour).Format(time.RFC3339)
	t2 := now.Add(-1 * time.Hour).Format(time.RFC3339)
	body := t1 + ` app.example.com 443 GET /a 200 0.010 1.1.1.1 127.0.0.1:8080 "a"` + "\n" +
		t2 + ` app.example.com 443 GET /b 403 0.010 2.2.2.2 127.0.0.1:8080 "b"` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	events := TailRecentEvents(path, nil, 30)
	if len(events) != 2 {
		t.Fatalf("got %d events", len(events))
	}
	if events[0].Status != 403 || events[1].Status != 200 {
		t.Fatalf("order/status: %+v", events)
	}
}
