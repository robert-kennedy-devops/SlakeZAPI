package usecase

import (
	"context"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type BillingUsecase struct {
	billing domain.BillingService
	log     *logger.Logger
}

func NewBillingUsecase(billing domain.BillingService, log *logger.Logger) *BillingUsecase {
	return &BillingUsecase{billing: billing, log: log}
}

func (u *BillingUsecase) GetUsage(ctx context.Context, tenantID string) (*domain.Usage, error) {
	return u.billing.GetUsage(ctx, tenantID)
}
