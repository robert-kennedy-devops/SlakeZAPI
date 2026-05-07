package queue

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// Job represents a unit of work to be processed by the pool.
type Job struct {
	ID      string
	Kind    string
	Payload interface{}
	Handler func(ctx context.Context, payload interface{}, attempt int) error
	Attempt int
}

// Pool is a bounded worker pool with retry and dead-letter support.
type Pool struct {
	jobs       chan Job
	deadLetter chan Job
	workerN    int
	maxRetries int
	log        *logger.Logger
	wg         sync.WaitGroup
	historyMu  sync.Mutex
	recent     []historyEntry
	deadRecent []historyEntry
	deadStore  map[string]Job
}

type historyEntry struct {
	ID        string
	Kind      string
	Attempt   int
	Status    string
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewPool creates and starts a worker pool.
func NewPool(workerN, bufferSize, maxRetries int, log *logger.Logger) *Pool {
	p := &Pool{
		jobs:       make(chan Job, bufferSize),
		deadLetter: make(chan Job, bufferSize/4+1),
		workerN:    workerN,
		maxRetries: maxRetries,
		log:        log,
		deadStore:  make(map[string]Job),
	}
	return p
}

// Start launches all workers. Call Stop to drain and shut down.
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workerN; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// dead-letter logger goroutine
	go p.logDeadLetters(ctx)
}

// Stop waits for all workers to finish.
func (p *Pool) Stop() {
	close(p.jobs)
	p.wg.Wait()
	close(p.deadLetter)
}

// Enqueue submits a job. Returns false if the queue is full.
func (p *Pool) Enqueue(job Job) bool {
	if job.Kind == "" {
		job.Kind = "generic"
	}
	p.track(job, "queued", "")
	select {
	case p.jobs <- job:
		return true
	default:
		p.track(job, "dropped", "queue full")
		p.log.Warn("queue full, dropping job", map[string]interface{}{"job_id": job.ID})
		return false
	}
}

func (p *Pool) Stats() (jobs int, deadLetters int, workers int) {
	p.historyMu.Lock()
	defer p.historyMu.Unlock()
	return len(p.jobs), len(p.deadStore), p.workerN
}

func (p *Pool) Snapshot() domain.QueueSnapshot {
	p.historyMu.Lock()
	defer p.historyMu.Unlock()

	recent := make([]domain.QueueJobView, 0, len(p.recent))
	for _, item := range p.recent {
		recent = append(recent, domain.QueueJobView{
			ID:        item.ID,
			Kind:      item.Kind,
			Attempt:   item.Attempt,
			Status:    item.Status,
			Error:     item.Error,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	dead := make([]domain.QueueJobView, 0, len(p.deadRecent))
	for _, item := range p.deadRecent {
		dead = append(dead, domain.QueueJobView{
			ID:        item.ID,
			Kind:      item.Kind,
			Attempt:   item.Attempt,
			Status:    item.Status,
			Error:     item.Error,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return domain.QueueSnapshot{
		Jobs:         len(p.jobs),
		DeadLetters:  len(p.deadStore),
		Workers:      p.workerN,
		Recent:       recent,
		DeadLettered: dead,
	}
}

func (p *Pool) DeadLetters(limit int) []domain.QueueJobView {
	p.historyMu.Lock()
	defer p.historyMu.Unlock()

	if limit <= 0 || limit > len(p.deadRecent) {
		limit = len(p.deadRecent)
	}
	items := make([]domain.QueueJobView, 0, limit)
	for _, item := range p.deadRecent[:limit] {
		items = append(items, domain.QueueJobView{
			ID:        item.ID,
			Kind:      item.Kind,
			Attempt:   item.Attempt,
			Status:    item.Status,
			Error:     item.Error,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return items
}

func (p *Pool) RequeueDeadLetter(id string) error {
	p.historyMu.Lock()
	job, ok := p.deadStore[id]
	if ok {
		delete(p.deadStore, id)
		for idx, item := range p.deadRecent {
			if item.ID == id {
				p.deadRecent = append(p.deadRecent[:idx], p.deadRecent[idx+1:]...)
				break
			}
		}
	}
	p.historyMu.Unlock()

	if !ok {
		return domain.ErrQueueJobNotFound
	}

	job.Attempt = 0
	if !p.Enqueue(job) {
		p.historyMu.Lock()
		p.deadStore[id] = job
		p.deadRecent = append([]historyEntry{{
			ID:        job.ID,
			Kind:      job.Kind,
			Attempt:   job.Attempt,
			Status:    "dead-letter",
			Error:     "queue full during requeue",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}}, p.deadRecent...)
		if len(p.deadRecent) > 25 {
			p.deadRecent = p.deadRecent[:25]
		}
		p.historyMu.Unlock()
		return fmt.Errorf("%w: queue full", domain.ErrConflict)
	}
	p.track(job, "requeued", "")
	return nil
}

// ─── internal ────────────────────────────────────────────────────────────────

func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	for job := range p.jobs {
		p.process(ctx, job)
	}
	_ = id
}

func (p *Pool) process(ctx context.Context, job Job) {
	attempt := job.Attempt + 1
	working := job
	working.Attempt = attempt
	p.track(working, "processing", "")
	err := job.Handler(ctx, job.Payload, attempt)
	if err == nil {
		p.track(working, "done", "")
		return
	}

	if attempt >= p.maxRetries {
		p.track(working, "dead-letter", err.Error())
		p.log.Error("job moved to dead-letter", map[string]interface{}{
			"job_id":  working.ID,
			"attempt": working.Attempt,
			"err":     err.Error(),
		})
		select {
		case p.deadLetter <- working:
		default:
		}
		return
	}

	// Exponential backoff: 1s, 2s, 4s …
	delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second

	p.log.Warn("job failed, retrying", map[string]interface{}{
		"job_id":  working.ID,
		"attempt": working.Attempt,
		"delay":   delay.String(),
		"err":     err.Error(),
	})
	p.track(working, "retrying", err.Error())

	time.AfterFunc(delay, func() {
		p.Enqueue(working)
	})
}

func (p *Pool) track(job Job, status, errText string) {
	p.historyMu.Lock()
	defer p.historyMu.Unlock()

	now := time.Now().UTC()
	entry := historyEntry{
		ID:        job.ID,
		Kind:      job.Kind,
		Attempt:   job.Attempt,
		Status:    status,
		Error:     errText,
		CreatedAt: now,
		UpdatedAt: now,
	}
	p.recent = append([]historyEntry{entry}, p.recent...)
	if len(p.recent) > 50 {
		p.recent = p.recent[:50]
	}
	if status == "dead-letter" {
		p.deadStore[job.ID] = job
		p.deadRecent = append([]historyEntry{entry}, p.deadRecent...)
		if len(p.deadRecent) > 25 {
			p.deadRecent = p.deadRecent[:25]
		}
	}
}

func (p *Pool) logDeadLetters(ctx context.Context) {
	for job := range p.deadLetter {
		p.log.Error("dead-letter job", map[string]interface{}{
			"job_id":  job.ID,
			"attempt": job.Attempt,
		})
		// Extension point: persist to DB, alert, re-queue manually, etc.
		_ = ctx
	}
}
