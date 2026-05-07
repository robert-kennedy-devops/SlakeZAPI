package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/whatsapp-saas/api/internal/billing"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/events"
	"github.com/whatsapp-saas/api/internal/observability"
	"github.com/whatsapp-saas/api/internal/queue"
	"github.com/whatsapp-saas/api/internal/repository"
	"github.com/whatsapp-saas/api/internal/testutil"
	"github.com/whatsapp-saas/api/internal/transport/ws"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type fakeWhatsAppService struct {
	status       domain.SessionStatus
	session      *domain.Session
	mediaContent []byte
}

func (f *fakeWhatsAppService) Connect(ctx context.Context, tenantID string) (string, error) {
	f.status = domain.SessionStatusConnecting
	f.session = &domain.Session{TenantID: tenantID, Status: f.status}
	return "qr-test", nil
}

func (f *fakeWhatsAppService) Disconnect(ctx context.Context, tenantID string) error {
	f.status = domain.SessionStatusDisconnected
	return nil
}

func (f *fakeWhatsAppService) Logout(ctx context.Context, tenantID string) error {
	f.status = domain.SessionStatusDisconnected
	f.session = nil
	return nil
}

func (f *fakeWhatsAppService) SendMessage(ctx context.Context, tenantID, phone, message string) (string, error) {
	return "wa-text-1", nil
}

func (f *fakeWhatsAppService) SendMediaMessage(ctx context.Context, tenantID string, req domain.SendMediaMessageRequest) (string, error) {
	return "wa-media-1", nil
}

func (f *fakeWhatsAppService) DownloadMedia(ctx context.Context, tenantID string, msg *domain.Message) (*domain.MediaDownload, error) {
	if len(f.mediaContent) == 0 {
		return nil, domain.ErrMessageMediaAbsent
	}
	return &domain.MediaDownload{
		FileName: msg.FileName,
		MimeType: msg.MimeType,
		Data:     f.mediaContent,
	}, nil
}

func (f *fakeWhatsAppService) GetSession(ctx context.Context, tenantID string) (*domain.Session, error) {
	if f.session == nil {
		return nil, domain.ErrSessionMetadataNotFound
	}
	return f.session, nil
}

func (f *fakeWhatsAppService) GetStatus(ctx context.Context, tenantID string) domain.SessionStatus {
	return f.status
}

func TestRouterBootstrapAndSendMessageFlow(t *testing.T) {
	db := testutil.OpenTestDB(t)
	server := newIntegrationServer(t, db)

	bootstrapResp := struct {
		TenantID string `json:"tenant_id"`
		APIKey   struct {
			APIKey string `json:"api_key"`
		} `json:"api_key"`
	}{}
	requestJSON(t, server, http.MethodPost, "/auth/bootstrap", "", map[string]string{
		"name":  "Tenant HTTP",
		"email": "http@example.com",
		"plan":  "growth",
	}, http.StatusCreated, &bootstrapResp)
	if bootstrapResp.TenantID == "" || bootstrapResp.APIKey.APIKey == "" {
		t.Fatalf("unexpected bootstrap response: %+v", bootstrapResp)
	}

	meResp := struct {
		Tenant *domain.Tenant `json:"tenant"`
		Usage  *domain.Usage  `json:"usage"`
	}{}
	requestJSON(t, server, http.MethodGet, "/auth/me", bootstrapResp.APIKey.APIKey, nil, http.StatusOK, &meResp)
	if meResp.Tenant == nil || meResp.Tenant.Email != "http@example.com" {
		t.Fatalf("unexpected /auth/me response: %+v", meResp)
	}

	sendResp := struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"`
	}{}
	requestJSON(t, server, http.MethodPost, "/messages/send", bootstrapResp.APIKey.APIKey, map[string]string{
		"phone":   "5511999999999",
		"message": "hello integration",
	}, http.StatusOK, &sendResp)
	if sendResp.MessageID == "" || sendResp.Status != "sent" {
		t.Fatalf("unexpected send response: %+v", sendResp)
	}

	listResp := make([]domain.Message, 0)
	requestJSON(t, server, http.MethodGet, "/messages", bootstrapResp.APIKey.APIKey, nil, http.StatusOK, &listResp)
	if len(listResp) != 1 {
		t.Fatalf("expected one message, got %d", len(listResp))
	}
	if listResp[0].Type != "text" || listResp[0].Body != "hello integration" {
		t.Fatalf("unexpected persisted message: %+v", listResp[0])
	}
}

func TestRouterDownloadsInboundMedia(t *testing.T) {
	db := testutil.OpenTestDB(t)
	server := newIntegrationServer(t, db)

	bootstrapResp := struct {
		TenantID string `json:"tenant_id"`
		APIKey   struct {
			APIKey string `json:"api_key"`
		} `json:"api_key"`
	}{}
	requestJSON(t, server, http.MethodPost, "/auth/bootstrap", "", map[string]string{
		"name":  "Tenant Media",
		"email": "media@example.com",
		"plan":  "growth",
	}, http.StatusCreated, &bootstrapResp)

	msgRepo := repository.NewMessageRepository(db)
	now := time.Now().UTC()
	msg := &domain.Message{
		ID:            "msg-media-1",
		TenantID:      bootstrapResp.TenantID,
		WhatsAppID:    "wa-inbound-media-1",
		Phone:         "5511999999999",
		Body:          "arquivo recebido",
		Type:          "document",
		MimeType:      "application/pdf",
		FileName:      "contrato.pdf",
		MediaURL:      "https://mmg.whatsapp.net/media",
		DirectPath:    "/v/t62.7118-24/example",
		FileLength:    4,
		MediaKey:      []byte("media-key"),
		FileSHA256:    []byte("file-sha"),
		FileEncSHA256: []byte("file-enc-sha"),
		Direction:     "inbound",
		Status:        domain.MessageStatusDelivered,
		SentAt:        now,
		CreatedAt:     now,
	}
	if err := msgRepo.Create(context.Background(), msg); err != nil {
		t.Fatalf("create inbound media message: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/messages/"+msg.ID+"/media", nil)
	req.Header.Set("Authorization", "Bearer "+bootstrapResp.APIKey.APIKey)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("unexpected content-type: %s", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got == "" {
		t.Fatal("expected content-disposition header")
	}
	if rec.Body.String() != "fake-media-bytes" {
		t.Fatalf("unexpected media body: %q", rec.Body.String())
	}
}

func TestRouterUserAuthSignUpLoginAndMe(t *testing.T) {
	db := testutil.OpenTestDB(t)
	server := newIntegrationServer(t, db)

	signUpResp := domain.AuthSessionResponse{}
	requestJSON(t, server, http.MethodPost, "/app/auth/signup", "", map[string]string{
		"name":        "Owner App",
		"email":       "owner@app.example",
		"password":    "supersecret123",
		"tenant_name": "Workspace App",
		"plan":        "starter",
	}, http.StatusCreated, &signUpResp)
	if signUpResp.Token == "" || signUpResp.User == nil || signUpResp.Tenant == nil || signUpResp.Membership == nil {
		t.Fatalf("unexpected signup response: %+v", signUpResp)
	}
	if signUpResp.Membership.Role != domain.UserRoleOwner {
		t.Fatalf("expected owner role, got %+v", signUpResp.Membership)
	}

	meResp := domain.CurrentUserResponse{}
	requestJSON(t, server, http.MethodGet, "/app/auth/me", signUpResp.Token, nil, http.StatusOK, &meResp)
	if meResp.User == nil || meResp.User.Email != "owner@app.example" {
		t.Fatalf("unexpected /app/auth/me response: %+v", meResp)
	}

	loginResp := domain.AuthSessionResponse{}
	requestJSON(t, server, http.MethodPost, "/app/auth/login", "", map[string]string{
		"email":    "owner@app.example",
		"password": "supersecret123",
	}, http.StatusOK, &loginResp)
	if loginResp.Token == "" || loginResp.User == nil || loginResp.User.ID != signUpResp.User.ID {
		t.Fatalf("unexpected login response: %+v", loginResp)
	}

	summaryResp := domain.TenantSummary{}
	requestJSONWithHeaders(t, server, http.MethodGet, "/app/tenant/summary", signUpResp.Token, nil, http.StatusOK, &summaryResp, map[string]string{
		"X-Tenant-ID": signUpResp.Tenant.ID,
	})
	if summaryResp.Tenant == nil || summaryResp.Tenant.ID != signUpResp.Tenant.ID {
		t.Fatalf("unexpected tenant summary: %+v", summaryResp)
	}
}

func TestRouterAppRoutesRespectUserRole(t *testing.T) {
	db := testutil.OpenTestDB(t)
	server := newIntegrationServer(t, db)

	signUpResp := domain.AuthSessionResponse{}
	requestJSON(t, server, http.MethodPost, "/app/auth/signup", "", map[string]string{
		"name":        "Viewer App",
		"email":       "viewer@app.example",
		"password":    "supersecret123",
		"tenant_name": "Workspace Viewer",
		"plan":        "growth",
	}, http.StatusCreated, &signUpResp)

	if _, err := db.Exec(`UPDATE tenant_users SET role = 'viewer' WHERE id = $1`, signUpResp.Membership.ID); err != nil {
		t.Fatalf("downgrade membership role: %v", err)
	}

	requestJSONWithHeaders(t, server, http.MethodGet, "/app/messages", signUpResp.Token, nil, http.StatusOK, &[]domain.Message{}, map[string]string{
		"X-Tenant-ID": signUpResp.Tenant.ID,
	})

	req := httptest.NewRequest(http.MethodPost, "/app/messages/send", bytes.NewReader([]byte(`{"phone":"5511999999999","message":"forbidden"}`)))
	req.Header.Set("Authorization", "Bearer "+signUpResp.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", signUpResp.Tenant.ID)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer send, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRouterCORSPreflight(t *testing.T) {
	db := testutil.OpenTestDB(t)
	server := newIntegrationServer(t, db)

	req := httptest.NewRequest(http.MethodOptions, "/app/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
}

func newIntegrationServer(t *testing.T, db *sql.DB) http.Handler {
	t.Helper()

	log := logger.New()
	eventBus := events.NewBus()
	metrics := observability.NewMetrics()
	pool := queue.NewPool(1, 10, 1, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		pool.Stop()
	})
	pool.Start(ctx)

	tenantRepo := repository.NewTenantRepository(db)
	userRepo := repository.NewUserRepository(db)
	tenantUserRepo := repository.NewTenantUserRepository(db)
	userSessionRepo := repository.NewUserSessionRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)
	msgRepo := repository.NewMessageRepository(db)
	webhookRepo := repository.NewWebhookRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	usageRepo := repository.NewUsageRepository(db)

	waSvc := &fakeWhatsAppService{
		status:       domain.SessionStatusConnected,
		mediaContent: []byte("fake-media-bytes"),
	}
	billingSvc := billing.NewService(usageRepo, subRepo, log)
	authUC := usecase.NewAuthUsecase(apiKeyRepo, tenantRepo, subRepo, billingSvc, waSvc, "test-salt", log)
	userAuthUC := usecase.NewUserAuthUsecase(userRepo, tenantRepo, tenantUserRepo, userSessionRepo, subRepo, log)
	msgUC := usecase.NewMessageUsecase(msgRepo, waSvc, billingSvc, eventBus, log)
	waUC := usecase.NewWhatsAppUsecase(waSvc, eventBus, log)
	webhookUC := usecase.NewWebhookUsecase(webhookRepo, subRepo, log)
	billingUC := usecase.NewBillingUsecase(billingSvc, log)
	hub := ws.NewHub(eventBus, log)

	return NewRouter(
		db,
		pool,
		authUC,
		userAuthUC,
		msgUC,
		waUC,
		webhookUC,
		billingUC,
		hub,
		metrics,
		time.Now().UTC(),
		1000,
		[]string{"http://localhost:3000"},
		log,
	)
}

func requestJSON(t *testing.T, handler http.Handler, method, path, apiKey string, body any, wantStatus int, out any) {
	t.Helper()
	requestJSONWithHeaders(t, handler, method, path, apiKey, body, wantStatus, out, nil)
}

func requestJSONWithHeaders(t *testing.T, handler http.Handler, method, path, apiKey string, body any, wantStatus int, out any, headers map[string]string) {
	t.Helper()

	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("unexpected status for %s %s: got %d want %d body=%s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("unmarshal response: %v body=%s", err, rec.Body.String())
		}
	}
}
