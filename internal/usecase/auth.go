package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type AuthUsecase struct {
	apiKeyRepo   domain.APIKeyRepository
	tenantRepo   domain.TenantRepository
	instanceRepo domain.InstanceRepository
	subRepo      domain.SubscriptionRepository
	billingSvc   domain.BillingService
	whatsapp     domain.WhatsAppService
	salt         string
	log          *logger.Logger
}

func NewAuthUsecase(
	apiKeyRepo domain.APIKeyRepository,
	tenantRepo domain.TenantRepository,
	instanceRepo domain.InstanceRepository,
	subRepo domain.SubscriptionRepository,
	billingSvc domain.BillingService,
	whatsapp domain.WhatsAppService,
	salt string,
	log *logger.Logger,
) *AuthUsecase {
	return &AuthUsecase{
		apiKeyRepo:   apiKeyRepo,
		tenantRepo:   tenantRepo,
		instanceRepo: instanceRepo,
		subRepo:      subRepo,
		billingSvc:   billingSvc,
		whatsapp:     whatsapp,
		salt:         salt,
		log:          log,
	}
}

func (u *AuthUsecase) BootstrapTenant(ctx context.Context, req domain.BootstrapTenantRequest) (*domain.BootstrapTenantResponse, error) {
	if req.Name == "" || req.Email == "" {
		return nil, fmt.Errorf("%w: name and email are required", domain.ErrBadRequest)
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, fmt.Errorf("%w: invalid email address", domain.ErrBadRequest)
	}

	if req.Plan == "" {
		req.Plan = domain.PlanStarter
	}
	plan, ok := domain.PlanByName(req.Plan)
	if !ok {
		return nil, fmt.Errorf("%w: invalid plan", domain.ErrBadRequest)
	}

	tenant := &domain.Tenant{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: time.Now().UTC(),
		Active:    true,
	}
	if err := u.tenantRepo.Create(ctx, tenant); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, err
	}

	sub := &domain.Subscription{
		ID:        uuid.NewString(),
		TenantID:  tenant.ID,
		PlanID:    plan.ID,
		Status:    "active",
		PeriodEnd: time.Now().UTC().Add(30 * 24 * time.Hour),
		CreatedAt: time.Now().UTC(),
	}
	if err := u.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}

	instance := &domain.Instance{
		ID:        uuid.NewString(),
		TenantID:  tenant.ID,
		Name:      "Principal",
		Status:    domain.SessionStatusDisconnected,
		IsDefault: true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := u.instanceRepo.Create(ctx, instance); err != nil {
		return nil, err
	}

	keyResp, err := u.CreateAPIKey(ctx, tenant.ID, "default")
	if err != nil {
		return nil, err
	}

	return &domain.BootstrapTenantResponse{
		TenantID: tenant.ID,
		Plan:     plan.Name,
		APIKey:   keyResp,
	}, nil
}

// CreateAPIKey generates a new API key for the given tenant.
func (u *AuthUsecase) CreateAPIKey(ctx context.Context, tenantID, label string) (*domain.CreateAPIKeyResponse, error) {
	// Verify tenant exists
	tenant, err := u.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !tenant.Active {
		return nil, domain.ErrTenantInactive
	}

	// Generate raw key
	rawKey, err := generateRandomKey(32)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	prefix := rawKey[:8]
	hash := hashKey(rawKey, u.salt)

	key := &domain.APIKey{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		KeyHash:   hash,
		KeyPrefix: prefix,
		Label:     label,
		Active:    true,
		CreatedAt: time.Now().UTC(),
		LastUsed:  time.Now().UTC(),
	}

	if err := u.apiKeyRepo.Create(ctx, key); err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("storing key: %w", err)
	}

	u.log.WithContext(ctx).Info("api key created", map[string]interface{}{
		"key_id": key.ID, "tenant_id": tenantID,
	})

	return &domain.CreateAPIKeyResponse{
		KeyID:  key.ID,
		APIKey: rawKey, // returned ONCE, never stored
		Prefix: prefix,
	}, nil
}

// ValidateAPIKey checks the key hash and returns the associated tenant ID.
func (u *AuthUsecase) ValidateAPIKey(ctx context.Context, rawKey string) (*domain.APIKey, error) {
	hash := hashKey(rawKey, u.salt)

	key, err := u.apiKeyRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, domain.ErrInvalidAPIKey
	}

	// Async update last_used (fire and forget, non-blocking)
	go func() {
		_ = u.apiKeyRepo.UpdateLastUsed(context.Background(), key.ID)
	}()

	return key, nil
}

func (u *AuthUsecase) ListAPIKeys(ctx context.Context, tenantID string) ([]domain.APIKey, error) {
	tenant, err := u.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !tenant.Active {
		return nil, domain.ErrTenantInactive
	}
	return u.apiKeyRepo.ListByTenant(ctx, tenantID)
}

func (u *AuthUsecase) RevokeAPIKey(ctx context.Context, tenantID, keyID string) error {
	key, err := u.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return domain.ErrInvalidAPIKey
	}
	if key.TenantID != tenantID {
		return domain.ErrUnauthorized
	}
	return u.apiKeyRepo.Revoke(ctx, keyID)
}

func (u *AuthUsecase) GetTenantSummary(ctx context.Context, tenantID string) (*domain.TenantSummary, error) {
	tenant, err := u.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var session *domain.Session
	instances, err := u.instanceRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(instances) > 0 {
		if current, err := u.whatsapp.GetSession(ctx, tenantID, instances[0].ID); err == nil {
			session = current
		}
	}

	usage, err := u.billingSvc.GetUsage(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var sub *domain.Subscription
	if current, err := u.subRepo.GetByTenant(ctx, tenantID); err == nil {
		sub = current
	}

	return &domain.TenantSummary{
		Tenant:    tenant,
		Session:   session,
		Instances: instances,
		Usage:     usage,
		Plan:      sub,
	}, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func generateRandomKey(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashKey(key, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt + key))
	return hex.EncodeToString(h.Sum(nil))
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
