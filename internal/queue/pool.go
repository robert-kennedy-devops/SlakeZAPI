package queue

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/whatsapp-saas/api/pkg/logger"
)

// Job represents a unit of work to be processed by the pool.
type Job struct {
	ID      string
	Payload interface{}
	Handler func(ctx context.Context, payload interface{}) error
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
}

// NewPool creates and starts a worker pool.
func NewPool(workerN, bufferSize, maxRetries int, log *logger.Logger) *Pool {
	p := &Pool{
		jobs:       make(chan Job, bufferSize),
		deadLetter: make(chan Job, bufferSize/4+1),
		workerN:    workerN,
		maxRetries: maxRetries,
		log:        log,
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
	select {
	case p.jobs <- job:
		return true
	default:
		p.log.Warn("queue full, dropping job", map[string]interface{}{"job_id": job.ID})
		return false
	}
}

func (p *Pool) Stats() (jobs int, deadLetters int, workers int) {
	return len(p.jobs), len(p.deadLetter), p.workerN
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
	err := job.Handler(ctx, job.Payload)
	if err == nil {
		return
	}

	job.Attempt++
	if job.Attempt >= p.maxRetries {
		p.log.Error("job moved to dead-letter", map[string]interface{}{
			"job_id":  job.ID,
			"attempt": job.Attempt,
			"err":     err.Error(),
		})
		select {
		case p.deadLetter <- job:
		default:
		}
		return
	}

	// Exponential backoff: 1s, 2s, 4s …
	delay := time.Duration(math.Pow(2, float64(job.Attempt))) * time.Second

	p.log.Warn("job failed, retrying", map[string]interface{}{
		"job_id":  job.ID,
		"attempt": job.Attempt,
		"delay":   delay.String(),
		"err":     err.Error(),
	})

	time.AfterFunc(delay, func() {
		p.Enqueue(job)
	})
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
