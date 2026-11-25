package scheduler

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

type Job struct {
	Name string
	Interval time.Duration
	IntervalOf func(ctx context.Context) time.Duration
	Timeout time.Duration
	Run func(ctx context.Context)
	RunErr func(ctx context.Context) error
	RetryMin time.Duration
	RetryMax time.Duration
}

type Scheduler struct {
	jobs []Job
	once sync.Once
	done chan struct{}
}

func New(jobs ...Job) *Scheduler {
	return &Scheduler{jobs: jobs, done: make(chan struct{})}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.once.Do(func() {
		var wg sync.WaitGroup
		for _, job := range s.jobs {
			wg.Add(1)
			go func(job Job) {
				defer wg.Done()
				s.runJob(ctx, job)
			}(job)
		}
		go func() { wg.Wait(); close(s.done) }()
	})
}

func (s *Scheduler) Wait(ctx context.Context) error {
	select {
	case <-s.done: return nil
	case <-ctx.Done(): return ctx.Err()
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	var failures int
	for ctx.Err() == nil {
		failed := false
		interval := job.Interval
		func() {
			defer func() {
				if p := recover(); p != nil {
					failed = true
					slog.Error("scheduled job panicked", "job", job.Name, "panic", p, "stack", string(debug.Stack()))
				}
			}()
			runCtx := ctx
			cancel := func() {}
			if job.Timeout > 0 { runCtx, cancel = context.WithTimeout(ctx, job.Timeout) }
			defer cancel()
			if job.RunErr != nil {
				if err := job.RunErr(runCtx); err != nil { failed = true; slog.Warn("scheduled job failed", "job", job.Name, "error", err) }
			} else if job.Run != nil { job.Run(runCtx) }
			if runCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil {
				failed = true
				slog.Warn("scheduled job deadline exceeded", "job", job.Name)
			}
			if ctx.Err() == nil && job.IntervalOf != nil {
				if d := job.IntervalOf(ctx); d > 0 { interval = d }
			}
		}()
		if ctx.Err() != nil { return }
		if interval <= 0 { interval = time.Minute }
		if failed { failures++; if job.RetryMin > 0 { interval = retryDelay(job.RetryMin, job.RetryMax, failures) } } else { failures = 0 }
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done(): timer.Stop(); return
		case <-timer.C:
		}
	}
}

func retryDelay(min, max time.Duration, failures int) time.Duration {
	if max < min { max = min }
	d := min
	for i := 1; i < failures && d < max; i++ {
		if d > max/2 { return max }
		d *= 2
	}
	return d
}
