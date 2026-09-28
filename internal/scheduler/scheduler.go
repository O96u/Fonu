package scheduler

import (
	"context"
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
		job.Run(ctx)
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
