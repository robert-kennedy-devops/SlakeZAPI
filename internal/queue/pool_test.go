package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

func TestPoolRequeueDeadLetter(t *testing.T) {
	log := logger.New()
	pool := NewPool(1, 10, 1, log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	defer pool.Stop()

	done := make(chan struct{}, 1)
	firstAttempt := true
	ok := pool.Enqueue(Job{
		ID:   "job-dead-1",
		Kind: "test",
		Handler: func(ctx context.Context, payload interface{}, attempt int) error {
			if firstAttempt {
				firstAttempt = false
				return errors.New("boom")
			}
			done <- struct{}{}
			return nil
		},
	})
	if !ok {
		t.Fatal("expected enqueue to succeed")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(pool.DeadLetters(10)) == 1 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if len(pool.DeadLetters(10)) != 1 {
		t.Fatalf("expected one dead-letter item, got %d", len(pool.DeadLetters(10)))
	}
	_, deadLetters, _ := pool.Stats()
	if deadLetters != 1 {
		t.Fatalf("expected Stats deadLetters=1, got %d", deadLetters)
	}

	if err := pool.RequeueDeadLetter("job-dead-1"); err != nil {
		t.Fatalf("requeue dead letter: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected requeued job to be processed")
	}

	if len(pool.DeadLetters(10)) != 0 {
		t.Fatalf("expected dead-letter store to be empty after requeue, got %d", len(pool.DeadLetters(10)))
	}
	_, deadLetters, _ = pool.Stats()
	if deadLetters != 0 {
		t.Fatalf("expected Stats deadLetters=0 after requeue, got %d", deadLetters)
	}
}

func TestPoolRequeueDeadLetterNotFound(t *testing.T) {
	pool := NewPool(1, 10, 1, logger.New())
	if err := pool.RequeueDeadLetter("missing"); !errors.Is(err, domain.ErrQueueJobNotFound) {
		t.Fatalf("expected ErrQueueJobNotFound, got %v", err)
	}
}
