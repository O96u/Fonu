package acme

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	certstore "github.com/fonu/fonu/internal/certificate"
)

type JobEvent struct {
	Type    string          `json:"type"`
	Level   string          `json:"level,omitempty"`
	Message string          `json:"message,omitempty"`
	Result  *JobDonePayload `json:"result,omitempty"`
}

type JobDonePayload struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	Domain    string `json:"domain,omitempty"`
	CertPath  string `json:"cert_path,omitempty"`
	KeyPath   string `json:"key_path,omitempty"`
	CertDir   string `json:"cert_dir,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type Job struct {
	id     string
	mu     sync.Mutex
	events []JobEvent
	done   bool
	waiters map[chan JobEvent]struct{}
}

type JobManager struct {
	mu   sync.Mutex
	jobs map[string]*Job
}

func NewJobManager() *JobManager {
	return &JobManager{jobs: map[string]*Job{}}
}

func (m *JobManager) Create() *Job {
	job := &Job{
		id:      fmt.Sprintf("%d", time.Now().UnixNano()),
		waiters: map[chan JobEvent]struct{}{},
	}
	m.mu.Lock()
	m.jobs[job.id] = job
	m.mu.Unlock()
	return job
}

func (m *JobManager) Get(id string) *Job {
	m.mu.Lock()
	job := m.jobs[id]
	m.mu.Unlock()
	return job
}

func (j *Job) ID() string {
	return j.id
}

func (j *Job) Append(level, message string) {
	j.emit(JobEvent{Type: "log", Level: level, Message: message})
}

func (j *Job) Info(message string)  { j.Append("info", message) }
func (j *Job) Warn(message string)  { j.Append("warn", message) }
func (j *Job) Error(message string) { j.Append("error", message) }

func (j *Job) Finish(result JobDonePayload) {
	j.emit(JobEvent{Type: "done", Result: &result})
}

func (j *Job) emit(ev JobEvent) {
	j.mu.Lock()
	j.events = append(j.events, ev)
	if ev.Type == "done" {
		j.done = true
	}
	for ch := range j.waiters {
		ch <- ev
	}
	j.mu.Unlock()
}

func (j *Job) Snapshot() []JobEvent {
	j.mu.Lock()
	out := append([]JobEvent(nil), j.events...)
	j.mu.Unlock()
	return out
}

func (j *Job) Subscribe() chan JobEvent {
	ch := make(chan JobEvent, 32)
	j.mu.Lock()
	j.waiters[ch] = struct{}{}
	j.mu.Unlock()
	return ch
}

func (j *Job) Unsubscribe(ch chan JobEvent) {
	j.mu.Lock()
	delete(j.waiters, ch)
	j.mu.Unlock()
	close(ch)
}

func (m *JobManager) Stream(ctx context.Context, w http.ResponseWriter, id string, flush func() error) error {
	job := m.Get(id)
	if job == nil {
		return fmt.Errorf("任务不存在或已过期")
	}

	ch := job.Subscribe()
	defer job.Unsubscribe(ch)

	for _, ev := range job.Snapshot() {
		if err := writeJobSSE(w, ev); err != nil {
			return err
		}
		if err := flush(); err != nil {
			return err
		}
		if ev.Type == "done" {
			return nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev := <-ch:
			if err := writeJobSSE(w, ev); err != nil {
				return err
			}
			if err := flush(); err != nil {
				return err
			}
			if ev.Type == "done" {
				return nil
			}
		}
	}
}

func writeJobSSE(w io.Writer, ev JobEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
	return err
}

func jobDoneFromRecord(rec certstore.Record, err error) JobDonePayload {
	if err != nil {
		return JobDonePayload{OK: false, Error: err.Error()}
	}
	payload := JobDonePayload{
		OK:       true,
		Domain:   rec.Domain,
		CertPath: rec.CertPath,
		KeyPath:  rec.KeyPath,
	}
	if rec.CertPath != "" {
		payload.CertDir = filepath.Dir(rec.CertPath)
	}
	if rec.ExpiresAt != nil {
		payload.ExpiresAt = rec.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return payload
}
