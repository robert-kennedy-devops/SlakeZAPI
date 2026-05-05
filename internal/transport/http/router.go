package http

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/observability"
	"github.com/whatsapp-saas/api/internal/queue"
	"github.com/whatsapp-saas/api/internal/transport/http/handler"
	"github.com/whatsapp-saas/api/internal/transport/ws"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// NewRouter builds the complete http.Handler tree.
func NewRouter(
	db *sql.DB,
	pool *queue.Pool,
	authUC *usecase.AuthUsecase,
	msgUC *usecase.MessageUsecase,
	waUC *usecase.WhatsAppUsecase,
	hookUC *usecase.WebhookUsecase,
	billingUC *usecase.BillingUsecase,
	hub *ws.Hub,
	metrics *observability.Metrics,
	startedAt time.Time,
	rateRPS int,
	log *logger.Logger,
) http.Handler {
	// ── Handlers ────────────────────────────────────────────────────────────
	authH := handler.NewAuthHandler(authUC, log)
	msgH := handler.NewMessageHandler(msgUC, log)
	waH := handler.NewWhatsAppHandler(waUC, log)
	hookH := handler.NewWebhookHandler(hookUC, billingUC, log)
	obsH := handler.NewObservabilityHandler(db, pool, metrics, startedAt)

	// ── Middleware chain builders ────────────────────────────────────────────
	withAuth := func(h http.HandlerFunc) http.Handler {
		return middleware.Auth(authUC, log)(
			middleware.RateLimit(rateRPS)(h),
		)
	}

	// ── Mux ─────────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("GET /health", obsH.Health)
	mux.HandleFunc("GET /readyz", obsH.Readyz)
	mux.HandleFunc("GET /livez", obsH.Livez)
	mux.Handle("GET /metrics", metrics.Handler())
	mux.HandleFunc("POST /auth/bootstrap", authH.Bootstrap)

	// Auth (requires existing API key to create a new one)
	mux.Handle("POST /auth/apikey", withAuth(authH.CreateAPIKey))
	mux.Handle("GET /auth/apikey", withAuth(authH.ListAPIKeys))
	mux.Handle("DELETE /auth/apikey/{id}", withAuth(authH.RevokeAPIKey))
	mux.Handle("GET /auth/me", withAuth(authH.Me))

	// WhatsApp
	mux.Handle("POST /whatsapp/connect", withAuth(waH.Connect))
	mux.Handle("GET /whatsapp/status", withAuth(waH.Status))
	mux.Handle("POST /whatsapp/disconnect", withAuth(waH.Disconnect))
	mux.Handle("POST /whatsapp/logout", withAuth(waH.Logout))

	// Messages
	mux.Handle("POST /messages/send", withAuth(msgH.Send))
	mux.Handle("POST /messages/send-media", withAuth(msgH.SendMedia))
	mux.Handle("GET /messages", withAuth(msgH.List))
	mux.Handle("GET /messages/{id}", withAuth(msgH.Get))
	mux.Handle("GET /messages/{id}/media", withAuth(msgH.GetMedia))

	// Webhooks
	mux.Handle("POST /webhook", withAuth(hookH.Register))
	mux.Handle("GET /webhook", withAuth(hookH.List))
	mux.Handle("DELETE /webhook/{id}", withAuth(hookH.Delete))
	mux.Handle("GET /usage", withAuth(hookH.Usage))

	// WebSocket
	mux.Handle("GET /ws", withAuth(hub.ServeHTTP))

	// ── Global middleware (applied outermost) ────────────────────────────────
	var root http.Handler = mux
	root = middleware.Logging(log)(root)
	root = middleware.Recover(log)(root)
	root = middleware.RequestID(root)
	root = metrics.Instrument(root)

	return root
}
