package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/whatsapp-saas/api/internal/audit"
	"github.com/whatsapp-saas/api/internal/billing"
	"github.com/whatsapp-saas/api/internal/config"
	"github.com/whatsapp-saas/api/internal/events"
	"github.com/whatsapp-saas/api/internal/observability"
	"github.com/whatsapp-saas/api/internal/queue"
	"github.com/whatsapp-saas/api/internal/repository"
	"github.com/whatsapp-saas/api/internal/transcription"
	transportHTTP "github.com/whatsapp-saas/api/internal/transport/http"
	"github.com/whatsapp-saas/api/internal/transport/ws"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/internal/webhook"
	"github.com/whatsapp-saas/api/internal/whatsapp"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// App holds all top-level dependencies.
type App struct {
	cfg        *config.Config
	server     *http.Server
	pool       *queue.Pool
	dispatcher *webhook.Dispatcher
	campaigns  *usecase.CampaignUsecase
	ops        *usecase.OperationsUsecase
	metrics    *observability.Metrics
	startedAt  time.Time
	log        *logger.Logger
}

// New wires the full dependency graph and returns a ready App.
func New(cfg *config.Config, log *logger.Logger) (*App, error) {
	startedAt := time.Now().UTC()

	// ── Database ────────────────────────────────────────────────────────────
	db, err := repository.NewPostgres(cfg.DatabaseURL, cfg.DBMaxConnections)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	// ── Repositories ────────────────────────────────────────────────────────
	tenantRepo := repository.NewTenantRepository(db)
	userRepo := repository.NewUserRepository(db)
	tenantUserRepo := repository.NewTenantUserRepository(db)
	userSessionRepo := repository.NewUserSessionRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	instanceRepo := repository.NewInstanceRepository(db)
	msgRepo := repository.NewMessageRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	webhookRepo := repository.NewWebhookRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	usageRepo := repository.NewUsageRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)

	auditSvc := audit.NewService(auditRepo, log)
	log = log.WithAuditSink(auditSvc)

	// ── Infrastructure ──────────────────────────────────────────────────────
	eventBus := events.NewBus()
	metrics := observability.NewMetrics()
	workerPool := queue.NewPool(cfg.WorkerCount, cfg.QueueBufferSize, cfg.WebhookRetries, log)

	// ── Services ────────────────────────────────────────────────────────────
	billingSvc := billing.NewService(usageRepo, subRepo, log)
	waMgr, err := whatsapp.NewManager(db, eventBus, instanceRepo, sessionRepo, msgRepo, billingSvc, log)
	if err != nil {
		return nil, fmt.Errorf("whatsapp manager: %w", err)
	}

	// ── Use Cases ───────────────────────────────────────────────────────────
	authUC := usecase.NewAuthUsecase(apiKeyRepo, tenantRepo, instanceRepo, subRepo, billingSvc, waMgr, cfg.APIKeySalt, log)
	userAuthUC := usecase.NewUserAuthUsecase(
		userRepo, tenantRepo, instanceRepo, tenantUserRepo, userSessionRepo, subRepo,
		cfg.UserAccessTokenTTL, cfg.UserRefreshTokenTTL, log,
	)
	msgUC := usecase.NewMessageUsecase(msgRepo, instanceRepo, campaignRepo, waMgr, billingSvc, eventBus, log)
	if transcriber := transcription.NewOpenAITranscriber(cfg.OpenAIAPIKey, cfg.OpenAITranscriptionModel); transcriber != nil {
		msgUC.SetAudioTranscriber(transcriber)
	}
	waUC := usecase.NewWhatsAppUsecase(waMgr, instanceRepo, eventBus, log)
	instanceUC := usecase.NewInstanceUsecase(tenantRepo, instanceRepo, log)
	billingUC := usecase.NewBillingUsecase(billingSvc, log)
	campaignUC := usecase.NewCampaignUsecase(campaignRepo, instanceRepo, msgUC, log)
	opsUC := usecase.NewOperationsUsecase(workerPool)
	auditUC := usecase.NewAuditUsecase(auditRepo)

	// ── WebSocket Hub ────────────────────────────────────────────────────────
	hub := ws.NewHub(eventBus, log)

	// ── Webhook Dispatcher ───────────────────────────────────────────────────
	dispatcher := webhook.NewDispatcher(
		webhookRepo, eventBus, workerPool,
		cfg.WebhookTimeout, cfg.WebhookRetries, log,
	)
	_ = dispatcher // started in Run()
	webhookUC := usecase.NewWebhookUsecase(webhookRepo, instanceRepo, subRepo, dispatcher, log)

	// ── HTTP Router ──────────────────────────────────────────────────────────
	router := transportHTTP.NewRouter(
		db, workerPool,
		authUC, userAuthUC, msgUC, waUC, webhookUC, billingUC,
		instanceUC, campaignUC,
		opsUC, auditUC,
		hub, metrics, startedAt,
		cfg.RateLimitRPS, cfg.AuthRateLimitRPM, cfg.MetricsToken,
		cfg.CORSAllowedOrigins,
		cfg.UserSessionCookieName, cfg.UserSessionCookieSecure, cfg.UserSessionCookieDomain, cfg.UserSessionCookieSameSite,
		cfg.BootstrapSecret, log,
	)

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		cfg:        cfg,
		server:     server,
		pool:       workerPool,
		dispatcher: dispatcher,
		campaigns:  campaignUC,
		ops:        opsUC,
		metrics:    metrics,
		startedAt:  startedAt,
		log:        log,
	}, nil
}

// Run starts the worker pool, webhook dispatcher, and HTTP server.
// It blocks until ctx is cancelled, then performs a graceful shutdown.
func (a *App) Run(ctx context.Context) error {
	// Start worker pool
	a.pool.Start(ctx)
	go a.dispatcher.Start(ctx)
	go a.runCampaignScheduler(ctx)

	// HTTP server in background
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("server starting", map[string]interface{}{"addr": a.server.Addr})
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Block until context cancelled or server error
	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	// Graceful shutdown
	a.log.Info("shutting down gracefully…")

	shutCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	a.pool.Stop()
	a.log.Info("server stopped")
	return nil
}

func (a *App) runCampaignScheduler(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.campaigns.RunDue(ctx); err != nil {
				a.log.Error("campaign scheduler failed", map[string]interface{}{"err": err.Error()})
			}
		}
	}
}
