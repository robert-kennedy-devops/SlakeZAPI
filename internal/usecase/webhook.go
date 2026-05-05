package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type WebhookUsecase struct {
	webhookRepo domain.WebhookRepository
	subRepo     domain.SubscriptionRepository
	log         *logger.Logger
}

func NewWebhookUsecase(
	webhookRepo domain.WebhookRepository,
	subRepo domain.SubscriptionRepository,
	log *logger.Logger,
) *WebhookUsecase {
	return &WebhookUsecase{
		webhookRepo: webhookRepo,
		subRepo:     subRepo,
		log:         log,
	}
}

// Register creates a new webhook for the tenant.
func (u *WebhookUsecase) Register(ctx context.Context, tenantID string, req domain.RegisterWebhookRequest) (*domain.Webhook, error) {
	// Check if tenant plan allows webhooks
	sub, err := u.subRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !sub.Plan.WebhookEnabled {
		return nil, domain.ErrBadRequest
	}

	secret, _ := generateSecret()

	wh := &domain.Webhook{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		URL:       req.URL,
		Events:    req.Events,
		Secret:    secret,
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}

	if err := u.webhookRepo.Create(ctx, wh); err != nil {
		return nil, err
	}

	u.log.WithContext(ctx).Info("webhook registered", map[string]interface{}{
		"webhook_id": wh.ID, "url": wh.URL,
	})

	return wh, nil
}

// ListByTenant returns active webhooks for a tenant.
func (u *WebhookUsecase) ListByTenant(ctx context.Context, tenantID string) ([]domain.Webhook, error) {
	return u.webhookRepo.GetByTenant(ctx, tenantID)
}

func (u *WebhookUsecase) Delete(ctx context.Context, tenantID, webhookID string) error {
	hooks, err := u.webhookRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		if hook.ID == webhookID {
			return u.webhookRepo.Delete(ctx, webhookID)
		}
	}
	return domain.ErrWebhookNotFound
}

func generateSecret() (string, error) {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
