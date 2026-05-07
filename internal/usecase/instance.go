package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type InstanceUsecase struct {
	tenantRepo   domain.TenantRepository
	instanceRepo domain.InstanceRepository
	log          *logger.Logger
}

func NewInstanceUsecase(
	tenantRepo domain.TenantRepository,
	instanceRepo domain.InstanceRepository,
	log *logger.Logger,
) *InstanceUsecase {
	return &InstanceUsecase{
		tenantRepo:   tenantRepo,
		instanceRepo: instanceRepo,
		log:          log,
	}
}

func (u *InstanceUsecase) EnsureDefault(ctx context.Context, tenantID string) (*domain.Instance, error) {
	instance, err := u.instanceRepo.GetDefaultByTenant(ctx, tenantID)
	if err == nil {
		return instance, nil
	}
	if err != domain.ErrInstanceNotFound {
		return nil, err
	}
	return u.Create(ctx, tenantID, domain.CreateInstanceRequest{Name: "Principal"})
}

func (u *InstanceUsecase) ListByTenant(ctx context.Context, tenantID string) ([]domain.Instance, error) {
	return u.instanceRepo.ListByTenant(ctx, tenantID)
}

func (u *InstanceUsecase) Create(ctx context.Context, tenantID string, req domain.CreateInstanceRequest) (*domain.Instance, error) {
	if _, err := u.tenantRepo.GetByID(ctx, tenantID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Instancia"
	}
	existing, err := u.instanceRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	instance := &domain.Instance{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		Name:      name,
		Status:    domain.SessionStatusDisconnected,
		IsDefault: len(existing) == 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := u.instanceRepo.Create(ctx, instance); err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}
	u.log.WithContext(ctx).Info("instance created", map[string]interface{}{"instance_id": instance.ID, "tenant_id": tenantID})
	return instance, nil
}
