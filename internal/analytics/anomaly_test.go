package analytics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestQueryAnomalyCounts5xxAgainstYesterday(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE access_facts (
		ts TEXT, client_ip TEXT, path TEXT, status INTEGER, request_ms REAL
	)`); err != nil {
		t.Fatal(err)
	}
	loc := time.Local
	now := time.Date(2026, 11, 2, 15, 0, 0, 0, loc)
	today := now.Add(-time.Hour).UTC().Format(time.RFC3339)
	yesterday := now.AddDate(0, 0, -1).UTC().Format(time.RFC3339)
	insert := `INSERT INTO access_facts(ts, client_ip, path, status, request_ms) VALUES (?, '1.1.1.1', ?, ?, ?)`
	if _, err := db.Exec(insert, today, "/admin", 500, 1200); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(insert, today, "/ok", 200, 20); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(insert, yesterday, "/admin", 500, 800); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(insert, yesterday, "/admin", 500, 800); err != nil {
		t.Fatal(err)
	}

	svc := &Service{db: db}
	out, err := svc.queryAnomaly(context.Background(), now, loc, "")
	if err != nil {
		t.Fatal(err)
	}
	if out.TotalRequests != 2 || int(out.Status5xx.Current) != 1 || int(out.Status5xx.Previous) != 2 {
		t.Fatalf("summary %+v total=%d", out.Status5xx, out.TotalRequests)
	}
	if len(out.Trend) != 7 {
		t.Fatalf("trend days %d", len(out.Trend))
	}
	if len(out.TopURLs) != 1 || out.TopURLs[0].Path != "/admin" || out.TopURLs[0].Errors5xxDelta != -1 {
		t.Fatalf("urls %+v", out.TopURLs)
	}
	if out.SlowRate.Current <= 0 {
		t.Fatalf("slow rate %+v", out.SlowRate)
	}
}

func TestQueryAnomalySkipsHeldConnections(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE access_facts (
		ts TEXT, client_ip TEXT, path TEXT, status INTEGER, request_ms REAL
	)`); err != nil {
		t.Fatal(err)
	}
	loc := time.Local
	now := time.Date(2026, 11, 2, 15, 0, 0, 0, loc)
	today := now.Add(-time.Hour).UTC().Format(time.RFC3339)
	yesterday := now.AddDate(0, 0, -1).UTC().Format(time.RFC3339)
	insert := `INSERT INTO access_facts(ts, client_ip, path, status, request_ms) VALUES (?, '1.1.1.1', ?, ?, ?)`
	rows := []struct {
		ts     string
		path   string
		status int
		ms     float64
	}{
		{today, "/ok", 200, 40},
		{today, "/slow", 200, 1500},
		{today, "/notifications/hub", 101, 5_000_000},
		{today, "-", 400, 30_000},
		{yesterday, "/ok", 200, 20},
	}
	for _, row := range rows {
		if _, err := db.Exec(insert, row.ts, row.path, row.status, row.ms); err != nil {
			t.Fatal(err)
		}
	}

	svc := &Service{db: db}
	out, err := svc.queryAnomaly(context.Background(), now, loc, "")
	if err != nil {
		t.Fatal(err)
	}
	if out.AvgRequestMS.Current < 700 || out.AvgRequestMS.Current > 800 {
		t.Fatalf("avg = %v, want about 770", out.AvgRequestMS.Current)
	}
	if out.AvgRequestMS.ChangeRate == nil || *out.AvgRequestMS.ChangeRate <= 0 {
		t.Fatalf("change %+v", out.AvgRequestMS)
	}
	if len(out.TopURLs) != 1 || out.TopURLs[0].Path != "/slow" {
		t.Fatalf("urls %+v", out.TopURLs)
	}
	if out.SlowRate.Current < 49 || out.SlowRate.Current > 51 {
		t.Fatalf("slow rate %v", out.SlowRate.Current)
	}
}

func TestAnomalyURLYesterdayMatchesTodayPathsOnly(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE access_facts (
		ts TEXT, client_ip TEXT, path TEXT, status INTEGER, request_ms REAL
	)`); err != nil {
		t.Fatal(err)
	}
	loc := time.Local
	now := time.Date(2026, 11, 2, 15, 0, 0, 0, loc)
	today := now.Add(-time.Hour).UTC().Format(time.RFC3339)
	yesterday := now.AddDate(0, 0, -1).UTC().Format(time.RFC3339)
	insert := `INSERT INTO access_facts(ts, client_ip, path, status, request_ms) VALUES (?, '1.1.1.1', ?, ?, ?)`
	rows := []struct {
		ts, path string
		status   int
		ms       float64
	}{
		{today, "/admin", 500, 1200},
		{yesterday, "/admin?from=1", 500, 900},
		{yesterday, "/other", 500, 900},
	}
	for _, row := range rows {
		if _, err := db.Exec(insert, row.ts, row.path, row.status, row.ms); err != nil {
			t.Fatal(err)
		}
	}

	svc := &Service{db: db}
	out, err := svc.queryAnomaly(context.Background(), now, loc, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.TopURLs) != 1 || out.TopURLs[0].Path != "/admin" || out.TopURLs[0].Errors5xxDelta != 0 {
		t.Fatalf("urls %+v", out.TopURLs)
	}
}
