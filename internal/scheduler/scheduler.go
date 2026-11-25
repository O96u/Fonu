package scheduler

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

type Job struct {
	Name       string
	Interval   time.Duration
	IntervalOf func(ctx context.Context) time.Duration
	Run        func(ctx context.Context)
}

type Scheduler struct {
	jobs []Job
}

func New(jobs ...Job) *Scheduler {
	return &Scheduler{jobs: jobs}
}

func (s *Scheduler) Start(ctx context.Context) {
	for _, job := range s.jobs {
		go s.runJob(ctx, job)
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	for {
		if ctx.Err() != nil { return }
		func() {
			defer func() {
				if p := recover(); p != nil { slog.Error("scheduled job panicked", "job", job.Name, "panic", p, "stack", string(debug.Stack())) }
			}()
			job.Run(ctx)
		}()
		interval := job.Interval
		if job.IntervalOf != nil {
			if d := job.IntervalOf(ctx); d > 0 {
				interval = d
			}
		}
		if interval <= 0 {
			interval = time.Minute
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
