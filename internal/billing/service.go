package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// Service implements domain.BillingService.
type Service struct {
	usageRepo domain.UsageRepository
	subRepo   domain.SubscriptionRepository
	log       *logger.Logger
}

func NewService(
	usageRepo domain.UsageRepository,
	subRepo domain.SubscriptionRepository,
	log *logger.Logger,
) *Service {
	return &Service{
		usageRepo: usageRepo,
		subRepo:   subRepo,
		log:       log,
	}
}

var _ domain.BillingService = (*Service)(nil)

// CheckLimit returns ErrLimitExceeded if the tenant has exceeded their plan quota.
func (s *Service) CheckLimit(ctx context.Context, tenantID string) error {
	sub, err := s.subRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		if err != domain.ErrNoSubscription {
			return err
		}
		plan, _ := domain.PlanByName(domain.PlanStarter)
		sub = &domain.Subscription{
			TenantID: tenantID,
			PlanID:   plan.ID,
			Plan:     &plan,
			Status:   "active",
		}
	}
	now := time.Now().UTC()
	if sub.TrialActive(now) {
		return nil
	}
	if sub.Status != "active" {
		return domain.ErrBillingCheckoutOnly
	}

	usage, err := s.usageRepo.GetCurrentMonth(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("billing.CheckLimit: %w", err)
	}

	if usage.Sent >= sub.Plan.MonthlyLimit {
		s.log.WithContext(ctx).Warn("monthly limit exceeded", map[string]interface{}{
			"sent":  usage.Sent,
			"limit": sub.Plan.MonthlyLimit,
			"plan":  sub.Plan.Name,
		})
		return domain.ErrLimitExceeded
	}

	return nil
}

// TrackSent increments the outbound message counter for the current month.
func (s *Service) TrackSent(ctx context.Context, tenantID string) error {
	return s.usageRepo.IncrementSent(ctx, tenantID)
}

// TrackReceived increments the inbound message counter for the current month.
func (s *Service) TrackReceived(ctx context.Context, tenantID string) error {
	return s.usageRepo.IncrementReceived(ctx, tenantID)
}

// GetUsage returns usage for the current calendar month.
func (s *Service) GetUsage(ctx context.Context, tenantID string) (*domain.Usage, error) {
	return s.usageRepo.GetCurrentMonth(ctx, tenantID)
}
