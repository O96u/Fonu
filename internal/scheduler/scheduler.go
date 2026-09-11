package scheduler

import (
	"context"
	"time"
)

type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context)
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
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()
	job.Run(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job.Run(ctx)
		}
	}
}
