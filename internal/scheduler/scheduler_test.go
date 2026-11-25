package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestJobDeadlineAndShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	s := New(Job{Name: "test", Timeout: 10*time.Millisecond, Interval: time.Hour, Run: func(ctx context.Context) {
		<-ctx.Done()
		done <- ctx.Err()
	}})
	s.Start(ctx)
	s.Start(ctx)
	select {
	case err := <-done:
		if err != context.DeadlineExceeded { t.Fatalf("job error: %v", err) }
	case <-time.After(time.Second): t.Fatal("job did not finish")
	}
	cancel()
	wait, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := s.Wait(wait); err != nil { t.Fatal(err) }
	select { case <-done: t.Fatal("job started twice"); default: }
}

func TestRetryDelayBounded(t *testing.T) {
	for i, want := range []time.Duration{time.Second, 2*time.Second, 4*time.Second, 5*time.Second, 5*time.Second} {
		if got := retryDelay(time.Second, 5*time.Second, i+1); got != want { t.Fatalf("retry %d: %v want %v", i+1, got, want) }
	}
}
