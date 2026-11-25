package analytics

import (
 "context"
 "os"
 "path/filepath"
 "testing"

 "github.com/fonu/fonu/internal/db"
)

func TestKPIBlockedTotalCountsEachEventOnce(t *testing.T) {
 conn, err := db.Open(filepath.Join(t.TempDir(), "kpi.db"))
 if err != nil { t.Fatal(err) }; defer conn.Close()
 schema, err := os.ReadFile(filepath.Join("..", "..", "migrations", "020_access_analytics.sql"))
 if err != nil { t.Fatal(err) }; if _, err = conn.Exec(string(schema)); err != nil { t.Fatal(err) }
 if _, err = conn.Exec("INSERT INTO access_facts(ts,client_ip,kind,status) VALUES ('2026-01-01T00:00:00Z','8.8.8.8','blocked',429)"); err != nil { t.Fatal(err) }
 kpi, funnel, err := New(conn,nil,nil).queryKPI(context.Background(),"2026-01-01T00:00:00Z","2026-01-01T23:59:59Z","")
 if err != nil { t.Fatal(err) }; if kpi.RiskRequests != 1 || funnel.BlockedTotal != 1 { t.Fatalf("unexpected totals: %+v %+v", kpi, funnel) }
}
