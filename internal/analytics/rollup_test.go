package analytics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fonu/fonu/internal/db"
)

func TestMinuteRollupExactBoundaries(t *testing.T) {
	ctx:=context.Background()
	conn,err:=db.Open(filepath.Join(t.TempDir(),"rollup.db")); if err!=nil {t.Fatal(err)}
	defer conn.Close()
	for _,name:=range []string{"020_access_analytics.sql","023_access_minute_rollups.sql"} {
		raw,err:=os.ReadFile(filepath.Join("..","..","migrations",name)); if err!=nil {t.Fatal(err)}
		if _,err:=conn.Exec(string(raw));err!=nil {t.Fatal(err)}
	}
	for _,r:=range []struct{ts,ip string; bytes int} {
		{"2026-01-01T00:00:04Z","1.2.3.4",9},
		{"2026-01-01T00:00:06Z","1.2.3.4",10},
		{"2026-01-01T00:01:12Z","1.2.3.4",20},
		{"2026-01-01T00:01:30Z","10.1.2.3",30},
		{"2026-01-01T00:02:03Z","8.8.8.8",40},
		{"2026-01-01T00:02:06Z","8.8.8.8",50},
	} { if _,err:=conn.Exec("INSERT INTO access_facts(ts,client_ip,bytes_sent) VALUES(?,?,?)",r.ts,r.ip,r.bytes);err!=nil {t.Fatal(err)} }
	s:=New(conn,nil,nil)
	start,_:=time.Parse(time.RFC3339,"2026-01-01T00:00:05Z")
	end,_:=time.Parse(time.RFC3339,"2026-01-01T00:02:05Z")
	check:=func(wantN,wantIPs int,wantBytes int64) {
		t.Helper()
		var n,ips int;var bytes int64
		if err:=s.querySeriesBucket(ctx,start,end,privateIPSQL,&n,&ips,&bytes);err!=nil {t.Fatal(err)}
		if n!=wantN||ips!=wantIPs||bytes!=wantBytes {t.Fatalf("got %d/%d/%d want %d/%d/%d",n,ips,bytes,wantN,wantIPs,wantBytes)}
	}
	check(3,2,70)
	if _,err:=conn.Exec("DELETE FROM access_facts WHERE ts='2026-01-01T00:01:12Z'");err!=nil {t.Fatal(err)}
	check(2,2,50)
}

func BenchmarkSeriesRawVsRollup(b *testing.B) {
	ctx:=context.Background()
	conn,err:=db.Open(filepath.Join(b.TempDir(),"benchmark.db")); if err!=nil {b.Fatal(err)}
	defer conn.Close()
	for _,name:=range []string{"020_access_analytics.sql","023_access_minute_rollups.sql"} {
		raw,err:=os.ReadFile(filepath.Join("..","..","migrations",name)); if err!=nil {b.Fatal(err)}
		if _,err:=conn.Exec(string(raw));err!=nil {b.Fatal(err)}
	}
	// Repeat 100 clients across each minute to simulate busy proxy traffic.
	_,err=conn.Exec(`WITH RECURSIVE n(i) AS (SELECT 0 UNION ALL SELECT i+1 FROM n WHERE i<99999)
		INSERT INTO access_facts(ts,client_ip,bytes_sent)
		SELECT strftime('%Y-%m-%dT%H:%M:%SZ','2026-01-01',printf('+%d seconds',i/20)),printf('8.8.8.%d',i%100),1024 FROM n`)
	if err!=nil {b.Fatal(err)}
	s:=New(conn,nil,nil)
	start:=time.Date(2026,1,1,0,0,0,0,time.UTC)
	end:=start.Add(2*time.Hour)
	var wantN,wantIPs int;var wantBytes int64
	rawQuery:=func(n,ips *int,bytes *int64)error {
		return conn.QueryRowContext(ctx,"SELECT COUNT(*),COUNT(DISTINCT client_ip),COALESCE(SUM(bytes_sent),0) FROM access_facts WHERE ts>=? AND ts<?",start.Format(time.RFC3339),end.Format(time.RFC3339)).Scan(n,ips,bytes)
	}
	if err:=rawQuery(&wantN,&wantIPs,&wantBytes);err!=nil {b.Fatal(err)}
	b.Run("raw",func(b *testing.B){
		b.ReportAllocs()
		for i:=0;i<b.N;i++ { var n,ips int; var bytes int64; if err:=rawQuery(&n,&ips,&bytes);err!=nil {b.Fatal(err)};if n!=wantN||ips!=wantIPs||bytes!=wantBytes {b.Fatal("raw mismatch")} }
	})
	b.Run("rollup",func(b *testing.B){
		b.ReportAllocs()
		for i:=0;i<b.N;i++ { var n,ips int;var bytes int64; if err:=s.querySeriesBucket(ctx,start,end,"",&n,&ips,&bytes);err!=nil {b.Fatal(err)};if n!=wantN||ips!=wantIPs||bytes!=wantBytes {b.Fatal("rollup mismatch")} }
	})
}
