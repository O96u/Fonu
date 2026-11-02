package analytics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/db"
)

func TestOverviewCachesFeed(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	schema, err := os.ReadFile(filepath.Join("..", "..", "migrations", "020_access_analytics.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	ts := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	if _, err := conn.Exec(`
		INSERT INTO access_facts(ts, client_ip, host, service, path, status, kind, geo_label, request_ms)
		VALUES (?, '1.2.3.4', 'ex.com', 'web', '/ok', 200, 'access', '中国', 12)
	`, ts); err != nil {
		t.Fatal(err)
	}

	svc := New(conn, nil, nil)
	ctx := context.Background()
	first, err := svc.Overview(ctx, "today")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Overview(ctx, "today")
	if err != nil {
		t.Fatal(err)
	}
	if first.UpdatedAt == "" || first.UpdatedAt != second.UpdatedAt {
		t.Fatalf("cache missed: %q vs %q", first.UpdatedAt, second.UpdatedAt)
	}
	if len(first.RecentEvents) != 1 || first.RecentEvents[0].Path != "/ok" {
		t.Fatalf("recent %+v", first.RecentEvents)
	}
	if len(first.ByService) != 1 || first.ByService[0].Service != "web" || first.ByService[0].Requests != 1 {
		t.Fatalf("services %+v", first.ByService)
	}
	if first.KPI.TotalRequests != 1 {
		t.Fatalf("kpi %+v", first.KPI)
	}
	yts := time.Now().UTC().Add(-26 * time.Hour).Format(time.RFC3339)
	if _, err := conn.Exec(`
		INSERT INTO access_facts(ts, client_ip, host, service, path, status, kind, geo_label, request_ms)
		VALUES (?, '5.6.7.8', 'ex.com', 'web', '/scan', 403, 'scan_suspected', '美国', 8)
	`, yts); err != nil {
		t.Fatal(err)
	}
	svc.overviewMu.Lock()
	svc.overviewCache = nil
	svc.overviewMu.Unlock()
	withYesterday, err := svc.Overview(ctx, "today")
	if err != nil {
		t.Fatal(err)
	}
	if withYesterday.KPI.YesterdayTotal != 1 || withYesterday.KPI.YesterdayUniqueIPs != 1 || withYesterday.KPI.YesterdayRisk != 1 {
		t.Fatalf("yesterday kpi %+v", withYesterday.KPI)
	}
	if withYesterday.Funnel.YesterdayScan != 1 || withYesterday.Funnel.ScanTrend == nil || *withYesterday.Funnel.ScanTrend != -100 {
		t.Fatalf("yesterday scan %+v", withYesterday.Funnel)
	}
	if withYesterday.KPI.UniqueIPTrend == nil || *withYesterday.KPI.UniqueIPTrend != 0 {
		t.Fatalf("unique trend %+v", withYesterday.KPI.UniqueIPTrend)
	}
}
