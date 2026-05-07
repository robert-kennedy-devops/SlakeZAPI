package usecase

import (
	"context"

	"github.com/whatsapp-saas/api/internal/domain"
)

type AuditUsecase struct {
	repo domain.AuditLogRepository
}

func NewAuditUsecase(repo domain.AuditLogRepository) *AuditUsecase {
	return &AuditUsecase{repo: repo}
}

func (u *AuditUsecase) List(ctx context.Context, tenantID, instanceID, actionPrefix string, limit int) ([]domain.AuditLog, error) {
	return u.repo.List(ctx, tenantID, instanceID, actionPrefix, limit)
}
