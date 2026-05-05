package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/whatsapp-saas/api/internal/observability"
	"github.com/whatsapp-saas/api/internal/queue"
	"github.com/whatsapp-saas/api/pkg/httputil"
)

type ObservabilityHandler struct {
	db        *sql.DB
	pool      *queue.Pool
	metrics   *observability.Metrics
	startedAt time.Time
}

func NewObservabilityHandler(db *sql.DB, pool *queue.Pool, metrics *observability.Metrics, startedAt time.Time) *ObservabilityHandler {
	return &ObservabilityHandler{
		db:        db,
		pool:      pool,
		metrics:   metrics,
		startedAt: startedAt,
	}
}

func (h *ObservabilityHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbReady := h.db.PingContext(ctx) == nil
	h.metrics.SetReadiness("database", dbReady)

	jobs, deadLetters, workers := h.pool.Stats()
	h.metrics.SetQueueDepth("jobs", jobs)
	h.metrics.SetQueueDepth("dead_letters", deadLetters)
	h.metrics.SetReadiness("worker_pool", workers > 0)

	status := http.StatusOK
	overall := "ok"
	if !dbReady {
		status = http.StatusServiceUnavailable
		overall = "degraded"
	}

	httputil.JSON(w, status, map[string]interface{}{
		"status":     overall,
		"time":       time.Now().UTC(),
		"uptime_sec": int(time.Since(h.startedAt).Seconds()),
		"components": map[string]interface{}{
			"database": map[string]bool{"ready": dbReady},
			"queue": map[string]int{
				"jobs":         jobs,
				"dead_letters": deadLetters,
				"workers":      workers,
			},
		},
	})
}

func (h *ObservabilityHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbReady := h.db.PingContext(ctx) == nil
	h.metrics.SetReadiness("database", dbReady)

	jobs, deadLetters, workers := h.pool.Stats()
	h.metrics.SetQueueDepth("jobs", jobs)
	h.metrics.SetQueueDepth("dead_letters", deadLetters)
	h.metrics.SetReadiness("worker_pool", workers > 0)

	if !dbReady || workers <= 0 {
		httputil.Error(w, http.StatusServiceUnavailable, "not ready")
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *ObservabilityHandler) Livez(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "alive"})
}
