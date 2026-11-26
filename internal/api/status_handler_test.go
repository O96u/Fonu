package api

import (
	"testing"
 "context"
 "sync/atomic"
 "strings"
 "os"
 "runtime"
 "net/http/httptest"
 "time"
 "strconv"
 "github.com/fonu/fonu/internal/config"

	certstore "github.com/fonu/fonu/internal/certificate"
)

func TestRequestTrendPercent(t *testing.T) {
	if got := requestTrendPercent(0, 0); got != nil {
		t.Fatalf("expected nil trend, got %v", *got)
	}
	if got := requestTrendPercent(10, 0); got == nil || *got != 100 {
		t.Fatalf("expected 100 for new traffic, got %v", got)
	}
	if got := requestTrendPercent(150, 100); got == nil || *got != 50 {
		t.Fatalf("expected 50, got %v", got)
	}
	if got := requestTrendPercent(50, 100); got == nil || *got != -50 {
		t.Fatalf("expected -50, got %v", got)
	}
}

func TestSummarizeCertificates(t *testing.T) {
	status, days, count, summary := summarizeCertificates(nil)
	if status != "none" || days != 0 || count != 0 || summary != "" {
		t.Fatalf("empty: got %s %d %d %q", status, days, count, summary)
	}

	ok := certstore.Record{Domain: "a.example.com", DaysLeft: 60, Status: "ok"}
	expiring := certstore.Record{Domain: "b.example.com", DaysLeft: 30, Status: "ok"}
	expired := certstore.Record{Domain: "c.example.com", DaysLeft: 0, Status: "ok"}
	failed := certstore.Record{Domain: "d.example.com", DaysLeft: 80, Status: "error"}

	status, days, count, _ = summarizeCertificates([]certstore.Record{ok})
	if status != "ok" || days != 60 || count != 1 {
		t.Fatalf("healthy: got %s %d %d", status, days, count)
	}

	status, days, count, _ = summarizeCertificates([]certstore.Record{expiring})
	if status != "expiring" || days != 30 || count != 1 {
		t.Fatalf("expiring: got %s %d %d", status, days, count)
	}

	status, days, count, _ = summarizeCertificates([]certstore.Record{ok, expiring})
	if status != "expiring" || days != 30 || count != 2 {
		t.Fatalf("mixed expiring: got %s %d %d", status, days, count)
	}

	status, _, _, _ = summarizeCertificates([]certstore.Record{expiring, expired})
	if status != "error" {
		t.Fatalf("expired should stay error, got %s", status)
	}

	status, _, _, _ = summarizeCertificates([]certstore.Record{failed, expiring})
	if status != "error" {
		t.Fatalf("apply failure should stay error, got %s", status)
	}
}

func TestParseProcessStatWithParentheses(t *testing.T){
 fields:=[]string{"S","1","2","3","4","5","6","7","8","9","10","31","12","13","14","15","16","17","18","999"}
 ticks,start,ok:=parseProcessStat("42 (name with ) chars) "+strings.Join(fields," "))
 if !ok||ticks!=43||start!=999{t.Fatalf("ticks=%d start=%d ok=%v",ticks,start,ok)}
 if _,_,ok:=parseProcessStat("broken");ok{t.Fatal("invalid stat accepted")}
}

func TestRuntimeSamplingSoak(t *testing.T){
 seconds,err:=strconv.Atoi(os.Getenv("FONU_SOAK_SECONDS"))
 if err!=nil||seconds<=0{t.Skip("set FONU_SOAK_SECONDS to enable")}
 if seconds>1800{t.Fatal("soak duration capped at 1800 seconds")}
 h:=NewStatusHandler(config.Config{DataDir:t.TempDir()},nil,nil,nil,nil,time.Now().UTC().Format(time.RFC3339),"")
 sample:=func(){r:=httptest.NewRequest("GET","/api/runtime",nil);w:=httptest.NewRecorder();h.Runtime(w,r);if w.Code!=200{t.Fatalf("runtime %d",w.Code)}}
 sample();runtime.GC()
 var before,after runtime.MemStats;runtime.ReadMemStats(&before)
 g0:=runtime.NumGoroutine();f0,_:=os.ReadDir("/proc/self/fd")
 deadline:=time.Now().Add(time.Duration(seconds)*time.Second);count:=0
 for time.Now().Before(deadline){sample();count++;time.Sleep(100*time.Millisecond)}
 runtime.GC();runtime.ReadMemStats(&after);f1,_:=os.ReadDir("/proc/self/fd")
 t.Logf("samples=%d heap_before=%d heap_after=%d goroutines_before=%d after=%d fd_before=%d after=%d",count,before.HeapAlloc,after.HeapAlloc,g0,runtime.NumGoroutine(),len(f0),len(f1))
 if runtime.NumGoroutine()>g0+4||len(f1)>len(f0)+4||after.HeapAlloc>before.HeapAlloc+8*1024*1024{t.Fatal("runtime sampling resource growth")}
}

func TestPublicIPProbeSurvivesShortDashboardDeadline(t *testing.T){
 h:=&StatusHandler{}
 var calls atomic.Int32
 release:=make(chan struct{})
 h.publicIPDetect=func(ctx context.Context)(string,string,error){calls.Add(1);select{case <-release:return "8.8.8.8","2606:4700::1111",nil;case <-ctx.Done():return "","",ctx.Err()}}
 ctx,cancel:=context.WithTimeout(context.Background(),10*time.Millisecond);defer cancel()
 _,_,at:=h.detectPublicIPs(ctx,false);if !at.IsZero(){t.Fatal("unfinished probe marked complete")}
 close(release)
 ctx2,cancel2:=context.WithTimeout(context.Background(),time.Second);defer cancel2()
 a,b,at:=h.detectPublicIPs(ctx2,false);if a!="8.8.8.8"||b!="2606:4700::1111"||at.IsZero()||calls.Load()!=1{t.Fatalf("probe result %q %q %v calls=%d",a,b,at,calls.Load())}
 _,_,_=h.detectPublicIPs(ctx2,true);if calls.Load()!=2{t.Fatal("forced refresh reused stale result")}
}
