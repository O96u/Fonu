package traffic

import (
	"testing"
 "context"
 "path/filepath"
 "github.com/fonu/fonu/internal/db"
	"time"

	"github.com/fonu/fonu/internal/proxy"
)

func TestCollectorAttributesConnectionsByHostAndPort(t *testing.T) {
	c := NewCollector(nil, "", nil)
	now := time.Now()
	line6893 := "2026-09-17T08:30:00+08:00 xxx.com 6893 GET /login 200 0.010 218.88.23.20 192.168.8.3:6893 100 200"
	line8443 := "2026-09-17T08:30:01+08:00 xxx.com 8443 GET / 200 0.010 218.88.23.20 192.168.8.3:8443 100 200"

	c.ingestLine(line6893, true)
	c.ingestLine(line8443, true)

	rules := []proxy.Rule{
		{
			ID:         1,
			ListenPort: 6893,
			Hosts:      []proxy.Host{{Hostname: "xxx.com"}},
		},
		{
			ID:         2,
			ListenPort: 8443,
			Hosts:      []proxy.Host{{Hostname: "xxx.com"}},
		},
	}

	stats := c.SnapshotForRules(rules)
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}
	byID := map[int64]RuleTraffic{}
	for _, s := range stats {
		byID[s.RuleID] = s
	}
	if byID[1].Connections != 1 {
		t.Fatalf("rule 6893 connections=%d want 1", byID[1].Connections)
	}
	if byID[2].Connections != 1 {
		t.Fatalf("rule 8443 connections=%d want 1", byID[2].Connections)
	}
	_ = now
}

func TestEndpointKeyIncludesPort(t *testing.T) {
	if got := EndpointKey("XXX.com", 6893); got != "xxx.com:6893" {
		t.Fatalf("got %q", got)
	}
}

func TestCollectorWaitAndFlush(t *testing.T){
 conn,err:=db.Open(filepath.Join(t.TempDir(),"traffic.db"));if err!=nil{t.Fatal(err)};defer conn.Close()
 if err:=db.Migrate(context.Background(),conn,filepath.Join("..","..","migrations"));err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background());defer cancel()
 c:=NewCollector(conn,"",nil)
 c.Start(ctx);c.Start(ctx)
 cancel()
 wait,stop:=context.WithTimeout(context.Background(),3*time.Second);defer stop()
 if err:=c.Wait(wait);err!=nil{t.Fatal(err)}
 c.ingestLine("2026-09-17T08:30:00+08:00 example.com 6893 GET / 200 0.010 8.8.8.8 192.168.8.3:6893 100 200",true)
 if err:=c.Flush(wait);err!=nil{t.Fatal(err)}
 totals,err:=c.store.LoadTotals(wait);if err!=nil{t.Fatal(err)}
 got:=totals[EndpointKey("example.com",6893)]
 if got.Upload!=100||got.Download!=200{t.Fatalf("final totals=%+v",got)}
}
