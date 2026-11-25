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
