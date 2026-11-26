package ingest

import (
	"testing"
	"time"
)

func TestRealtimeRingAggregatesAndExpires(t *testing.T) {
	now:=time.Unix(1800000000,0)
	r:=NewRealtimeRing(60*time.Second)
	for i:=0;i<60000;i++ {
		r.addAt(FactRow{TS:now.Add(-time.Duration(i%60)*time.Second),BytesSent:1024,RequestMS:5},now)
	}
	r.addAt(FactRow{TS:now.Add(-time.Minute-time.Second),BytesSent:100,RequestMS:100},now)
	r.addAt(FactRow{TS:now.Add(time.Second),BytesSent:100,RequestMS:100},now)
	got:=r.Snapshot(now)
	if got.QPS!=1000 || got.TrafficKBps!=1000 || got.AvgUpstreamMS!=5 { t.Fatalf("unexpected stats: %+v",got) }
	if len(r.samples)>60 { t.Fatalf("unbounded storage: %d",len(r.samples)) }
	if got:=r.Snapshot(now.Add(61*time.Second));got.QPS!=0 || got.TrafficKBps!=0 || got.AvgUpstreamMS!=0 { t.Fatalf("expired stats: %+v",got) }
}

func BenchmarkRealtimeRing(b *testing.B) {
	now:=time.Now().Truncate(time.Second)
	r:=NewRealtimeRing(time.Minute)
	row:=FactRow{TS:now,BytesSent:1024,RequestMS:10}
	b.ReportAllocs()
	b.ResetTimer()
	for i:=0;i<b.N;i++ { r.addAt(row,now) }
}
