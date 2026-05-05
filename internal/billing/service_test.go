package billing

import (
	"context"
	"testing"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type usageRepoStub struct {
	usage *domain.Usage
}

func (u *usageRepoStub) IncrementSent(ctx context.Context, tenantID string) error     { return nil }
func (u *usageRepoStub) IncrementReceived(ctx context.Context, tenantID string) error { return nil }
func (u *usageRepoStub) GetCurrentMonth(ctx context.Context, tenantID string) (*domain.Usage, error) {
	return u.usage, nil
}

type subRepoStub struct {
	sub *domain.Subscription
	err error
}

func (s *subRepoStub) GetByTenant(ctx context.Context, tenantID string) (*domain.Subscription, error) {
	return s.sub, s.err
}

func (s *subRepoStub) Upsert(ctx context.Context, sub *domain.Subscription) error { return nil }

func TestCheckLimitFallsBackToStarterPlan(t *testing.T) {
	service := NewService(
		&usageRepoStub{usage: &domain.Usage{TenantID: "tenant-1", Sent: 999}},
		&subRepoStub{err: domain.ErrNoSubscription},
		logger.New(),
	)

	if err := service.CheckLimit(context.Background(), "tenant-1"); err != nil {
		t.Fatalf("expected starter fallback to allow usage below limit, got %v", err)
	}
}

func TestCheckLimitReturnsLimitExceeded(t *testing.T) {
	plan, _ := domain.PlanByName(domain.PlanGrowth)
	service := NewService(
		&usageRepoStub{usage: &domain.Usage{TenantID: "tenant-1", Sent: plan.MonthlyLimit}},
		&subRepoStub{sub: &domain.Subscription{TenantID: "tenant-1", Plan: &plan}},
		logger.New(),
	)

	if err := service.CheckLimit(context.Background(), "tenant-1"); err != domain.ErrLimitExceeded {
		t.Fatalf("expected ErrLimitExceeded, got %v", err)
	}
}
