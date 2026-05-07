package http

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
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
	userAuthUC *usecase.UserAuthUsecase,
	msgUC *usecase.MessageUsecase,
	waUC *usecase.WhatsAppUsecase,
	hookUC *usecase.WebhookUsecase,
	billingUC *usecase.BillingUsecase,
	instanceUC *usecase.InstanceUsecase,
	campaignUC *usecase.CampaignUsecase,
	opsUC *usecase.OperationsUsecase,
	auditUC *usecase.AuditUsecase,
	hub *ws.Hub,
	metrics *observability.Metrics,
	startedAt time.Time,
	rateRPS int,
	corsAllowedOrigins []string,
	userSessionCookieName string,
	userSessionCookieSecure bool,
	userSessionCookieDomain string,
	userSessionCookieSameSite string,
	log *logger.Logger,
) http.Handler {
	// ── Handlers ────────────────────────────────────────────────────────────
	authH := handler.NewAuthHandler(authUC, log)
	userAuthH := handler.NewUserAuthHandler(
		userAuthUC,
		metrics,
		userSessionCookieName,
		userSessionCookieSecure,
		userSessionCookieDomain,
		userSessionCookieSameSite,
		log,
	)
	msgH := handler.NewMessageHandler(msgUC, log)
	waH := handler.NewWhatsAppHandler(waUC, log)
	hookH := handler.NewWebhookHandler(hookUC, billingUC, log)
	instanceH := handler.NewInstanceHandler(instanceUC, log)
	campaignH := handler.NewCampaignHandler(campaignUC, log)
	opsH := handler.NewOperationsHandler(opsUC)
	auditH := handler.NewAuditHandler(auditUC)
	docsH := handler.NewDocsHandler()
	obsH := handler.NewObservabilityHandler(db, pool, metrics, startedAt)

	// ── Middleware chain builders ────────────────────────────────────────────
	withAuth := func(h http.HandlerFunc) http.Handler {
		return middleware.Auth(authUC, log)(
			middleware.RateLimit(rateRPS)(h),
		)
	}
	withUserAuth := func(h http.HandlerFunc) http.Handler {
		return middleware.UserAuth(userAuthUC, log)(h)
	}
	withUserRole := func(h http.HandlerFunc, roles ...domain.UserRole) http.Handler {
		return middleware.UserAuth(userAuthUC, log)(
			middleware.RequireUserRole(roles...)(h),
		)
	}

	// ── Mux ─────────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("GET /health", obsH.Health)
	mux.HandleFunc("GET /readyz", obsH.Readyz)
	mux.HandleFunc("GET /livez", obsH.Livez)
	mux.Handle("GET /metrics", metrics.Handler())
	mux.HandleFunc("GET /docs", docsH.Index)
	mux.HandleFunc("GET /docs/{name}", docsH.Serve)
	mux.HandleFunc("POST /auth/bootstrap", authH.Bootstrap)
	mux.HandleFunc("POST /app/auth/signup", userAuthH.SignUp)
	mux.HandleFunc("POST /app/auth/login", userAuthH.Login)
	mux.HandleFunc("POST /app/auth/refresh", userAuthH.Refresh)
	mux.Handle("POST /app/auth/logout", withUserAuth(userAuthH.Logout))
	mux.Handle("GET /app/auth/me", withUserAuth(userAuthH.Me))
	mux.Handle("GET /app/tenant/summary", withUserRole(authH.Me, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/instances", withUserRole(instanceH.List, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("POST /app/instances", withUserRole(instanceH.Create, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("GET /app/members", withUserRole(userAuthH.ListMembers, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("POST /app/members", withUserRole(userAuthH.AddMember, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("POST /app/members/{id}/role", withUserRole(userAuthH.UpdateMemberRole, domain.UserRoleOwner, domain.UserRoleAdmin))

	// App dashboard routes (user session based)
	mux.Handle("GET /app/apikeys", withUserRole(authH.ListAPIKeys, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("POST /app/apikeys", withUserRole(authH.CreateAPIKey, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("DELETE /app/apikeys/{id}", withUserRole(authH.RevokeAPIKey, domain.UserRoleOwner, domain.UserRoleAdmin))

	mux.Handle("POST /app/whatsapp/connect", withUserRole(waH.Connect, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("GET /app/whatsapp/status", withUserRole(waH.Status, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/whatsapp/qr", withUserRole(waH.QRPage, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("GET /app/whatsapp/qr.png", withUserRole(waH.QRPNG, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("POST /app/whatsapp/disconnect", withUserRole(waH.Disconnect, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("POST /app/whatsapp/logout", withUserRole(waH.Logout, domain.UserRoleOwner, domain.UserRoleAdmin))

	mux.Handle("POST /app/messages/send", withUserRole(msgH.Send, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("POST /app/messages/send-bulk", withUserRole(msgH.SendBulk, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("POST /app/messages/send-media", withUserRole(msgH.SendMedia, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("POST /app/messages/send-interactive", withUserRole(msgH.SendInteractive, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("POST /app/messages/send-group", withUserRole(msgH.SendGroup, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("POST /app/status/post", withUserRole(msgH.StatusPost, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("POST /app/contacts/resolve", withUserRole(msgH.ResolveContacts, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/messages", withUserRole(msgH.List, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/conversations", withUserRole(msgH.ListConversations, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("POST /app/conversations/{phone}", withUserRole(msgH.UpdateConversation, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("GET /app/groups", withUserRole(msgH.ListGroups, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/messages/{id}", withUserRole(msgH.Get, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/messages/{id}/media", withUserRole(msgH.GetMedia, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/campaigns", withUserRole(campaignH.List, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("POST /app/campaigns", withUserRole(campaignH.Create, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("POST /app/campaigns/{id}/run", withUserRole(campaignH.Run, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))

	mux.Handle("POST /app/webhooks", withUserRole(hookH.Register, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("GET /app/webhooks", withUserRole(hookH.List, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("DELETE /app/webhooks/{id}", withUserRole(hookH.Delete, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("GET /app/webhooks/deliveries", withUserRole(hookH.ListDeliveries, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("POST /app/webhooks/deliveries/{id}/replay", withUserRole(hookH.ReplayDelivery, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("GET /app/usage", withUserRole(hookH.Usage, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/queue", withUserRole(opsH.Queue, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("GET /app/queue/dead-letters", withUserRole(opsH.DeadLetters, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))
	mux.Handle("POST /app/queue/dead-letters/{id}/requeue", withUserRole(opsH.RequeueDeadLetter, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator))
	mux.Handle("GET /app/audit", withUserRole(auditH.List, domain.UserRoleOwner, domain.UserRoleAdmin))
	mux.Handle("GET /app/ws", withUserRole(hub.ServeHTTP, domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer))

	// Auth (requires existing API key to create a new one)
	mux.Handle("POST /auth/apikey", withAuth(authH.CreateAPIKey))
	mux.Handle("GET /auth/apikey", withAuth(authH.ListAPIKeys))
	mux.Handle("DELETE /auth/apikey/{id}", withAuth(authH.RevokeAPIKey))
	mux.Handle("GET /auth/me", withAuth(authH.Me))

	// WhatsApp
	mux.Handle("POST /whatsapp/connect", withAuth(waH.Connect))
	mux.Handle("GET /whatsapp/status", withAuth(waH.Status))
	mux.Handle("GET /whatsapp/qr", withAuth(waH.QRPage))
	mux.Handle("GET /whatsapp/qr.png", withAuth(waH.QRPNG))
	mux.Handle("POST /whatsapp/disconnect", withAuth(waH.Disconnect))
	mux.Handle("POST /whatsapp/logout", withAuth(waH.Logout))

	// Messages
	mux.Handle("POST /messages/send", withAuth(msgH.Send))
	mux.Handle("POST /messages/send-bulk", withAuth(msgH.SendBulk))
	mux.Handle("POST /messages/send-media", withAuth(msgH.SendMedia))
	mux.Handle("POST /messages/send-interactive", withAuth(msgH.SendInteractive))
	mux.Handle("POST /messages/send-group", withAuth(msgH.SendGroup))
	mux.Handle("POST /status/post", withAuth(msgH.StatusPost))
	mux.Handle("POST /contacts/resolve", withAuth(msgH.ResolveContacts))
	mux.Handle("GET /messages", withAuth(msgH.List))
	mux.Handle("GET /conversations", withAuth(msgH.ListConversations))
	mux.Handle("POST /conversations/{phone}", withAuth(msgH.UpdateConversation))
	mux.Handle("GET /groups", withAuth(msgH.ListGroups))
	mux.Handle("GET /messages/{id}", withAuth(msgH.Get))
	mux.Handle("GET /messages/{id}/media", withAuth(msgH.GetMedia))
	mux.Handle("GET /instances", withAuth(instanceH.List))
	mux.Handle("POST /instances", withAuth(instanceH.Create))
	mux.Handle("GET /campaigns", withAuth(campaignH.List))
	mux.Handle("POST /campaigns", withAuth(campaignH.Create))
	mux.Handle("POST /campaigns/{id}/run", withAuth(campaignH.Run))

	// Webhooks
	mux.Handle("POST /webhook", withAuth(hookH.Register))
	mux.Handle("GET /webhook", withAuth(hookH.List))
	mux.Handle("DELETE /webhook/{id}", withAuth(hookH.Delete))
	mux.Handle("GET /webhook/deliveries", withAuth(hookH.ListDeliveries))
	mux.Handle("POST /webhook/deliveries/{id}/replay", withAuth(hookH.ReplayDelivery))
	mux.Handle("GET /usage", withAuth(hookH.Usage))
	mux.Handle("GET /queue", withAuth(opsH.Queue))
	mux.Handle("GET /queue/dead-letters", withAuth(opsH.DeadLetters))
	mux.Handle("POST /queue/dead-letters/{id}/requeue", withAuth(opsH.RequeueDeadLetter))
	mux.Handle("GET /audit", withAuth(auditH.List))

	// WebSocket
	mux.Handle("GET /ws", withAuth(hub.ServeHTTP))

	// ── Global middleware (applied outermost) ────────────────────────────────
	var root http.Handler = mux
	root = middleware.Logging(log)(root)
	root = middleware.Recover(log)(root)
	root = middleware.RequestID(root)
	root = middleware.CORS(corsAllowedOrigins)(root)
	root = metrics.Instrument(root)

	return root
}
