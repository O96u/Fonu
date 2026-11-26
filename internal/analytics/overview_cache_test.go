package analytics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
 "reflect"
 "github.com/fonu/fonu/internal/awareness"
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

func TestSummaryOverviewPreservesDisplayedStatistics(t *testing.T){
 conn,err:=db.Open(filepath.Join(t.TempDir(),"summary.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 ctx:=context.Background()
 if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 now:=time.Now().UTC().Truncate(time.Second)
 for i:=0;i<40;i++{
  ts:=now.Add(-time.Duration(i)*12*time.Hour).Format(time.RFC3339)
  if _,err:=conn.Exec("INSERT INTO access_facts(ts,client_ip,service,geo_country,status,kind,bytes_sent) VALUES(?,'8.8.8.8','test','中国',403,'blocked',8)",ts);err!=nil{t.Fatal(err)}
 }
 s:=New(conn,nil,nil)
 for _,r:=range []string{awareness.Range30d,awareness.RangeAll}{
  full,err:=s.Overview(ctx,r);if err!=nil{t.Fatal(err)}
  summary,err:=s.OverviewFor(ctx,r,true);if err!=nil{t.Fatal(err)}
  if !reflect.DeepEqual(full.KPI,summary.KPI)||!reflect.DeepEqual(full.Funnel,summary.Funnel)||!reflect.DeepEqual(full.Series,summary.Series)||!reflect.DeepEqual(full.ByCountry,summary.ByCountry)||!reflect.DeepEqual(full.ByService,summary.ByService){t.Fatalf("summary mismatch %s",r)}
  if len(summary.RecentEvents)!=5||len(summary.Anomaly.Trend)!=0||len(full.Anomaly.Trend)==0{t.Fatalf("incorrect scoped data %s",r)}
 }
}
func BenchmarkDashboardOverview30Days(b *testing.B){
 ctx:=context.Background()
 conn,err:=db.Open(filepath.Join(b.TempDir(),"overview.db"));if err!=nil{b.Fatal(err)};defer conn.Close()
 if err:=db.Migrate(ctx,conn,filepath.Join("..","..","migrations"));err!=nil{b.Fatal(err)}
 now:=time.Now().UTC().Truncate(time.Second)
 _,err=conn.Exec(`WITH RECURSIVE n(i) AS (SELECT 0 UNION ALL SELECT i+1 FROM n WHERE i<99999)
 INSERT INTO access_facts(ts,client_ip,service,geo_country,status,bytes_sent,user_agent,path)
 SELECT strftime('%Y-%m-%dT%H:%M:%SZ',?,printf('-%d seconds',i*25)),printf('8.8.8.%d',i%100),printf('service%d',i%10),'中国',200,1024,printf('client%d',i%30),'/ok' FROM n`,now.Format(time.RFC3339))
 if err!=nil{b.Fatal(err)}
 s:=New(conn,nil,nil);loc:=time.UTC
 for _,compact:=range []bool{false,true}{
  name:="full";if compact{name="summary"}
  b.Run(name,func(b *testing.B){b.ReportAllocs();for i:=0;i<b.N;i++{if _,err:=s.computeOverview(ctx,awareness.Range30d,loc,"UTC",false,compact);err!=nil{b.Fatal(err)}}})
 }
}
