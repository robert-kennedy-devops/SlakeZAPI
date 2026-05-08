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

type BillingUsecase struct {
	billing    domain.BillingService
	tenantRepo domain.TenantRepository
	subRepo    domain.SubscriptionRepository
	gateway    domain.BillingGateway
	appBaseURL string
	log        *logger.Logger
}

func NewBillingUsecase(
	billing domain.BillingService,
	tenantRepo domain.TenantRepository,
	subRepo domain.SubscriptionRepository,
	gateway domain.BillingGateway,
	appBaseURL string,
	log *logger.Logger,
) *BillingUsecase {
	return &BillingUsecase{
		billing:    billing,
		tenantRepo: tenantRepo,
		subRepo:    subRepo,
		gateway:    gateway,
		appBaseURL: strings.TrimRight(appBaseURL, "/"),
		log:        log,
	}
}

func (u *BillingUsecase) GetUsage(ctx context.Context, tenantID string) (*domain.Usage, error) {
	return u.billing.GetUsage(ctx, tenantID)
}

func (u *BillingUsecase) GetSubscription(ctx context.Context, tenantID string) (*domain.Subscription, error) {
	return u.subRepo.GetByTenant(ctx, tenantID)
}

func (u *BillingUsecase) CreateCheckout(ctx context.Context, tenantID string, req domain.CreateCheckoutRequest) (*domain.BillingActionResponse, error) {
	if u.gateway == nil {
		return nil, domain.ErrBillingNotConfigured
	}

	plan, err := u.requirePaidPlan(req.Plan)
	if err != nil {
		return nil, err
	}

	tenant, err := u.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	session, err := u.gateway.CreateCheckoutSession(ctx, domain.CheckoutSessionRequest{
		TenantID:      tenant.ID,
		TenantName:    tenant.Name,
		CustomerEmail: tenant.Email,
		Plan:          plan,
		SuccessURL:    u.appURL("/dashboard?billing=success"),
		CancelURL:     u.appURL("/dashboard?billing=cancelled"),
	})
	if err != nil {
		return nil, err
	}

	return &domain.BillingActionResponse{
		CheckoutURL:      session.URL,
		Provider:         session.Provider,
		RequiresCheckout: true,
		Message:          "checkout created",
	}, nil
}

func (u *BillingUsecase) OpenPortal(ctx context.Context, tenantID string) (*domain.BillingActionResponse, error) {
	if u.gateway == nil {
		return nil, domain.ErrBillingNotConfigured
	}

	sub, err := u.subRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sub.ProviderCustomerID) == "" {
		return nil, fmt.Errorf("%w: no billing customer is linked yet", domain.ErrBadRequest)
	}

	session, err := u.gateway.CreateBillingPortalSession(ctx, sub.ProviderCustomerID, u.appURL("/dashboard?billing=portal"))
	if err != nil {
		return nil, err
	}

	return &domain.BillingActionResponse{
		PortalURL: session.URL,
		Provider:  session.Provider,
		Message:   "portal created",
	}, nil
}

func (u *BillingUsecase) ChangePlan(ctx context.Context, tenantID string, req domain.ChangeSubscriptionPlanRequest) (*domain.BillingActionResponse, error) {
	plan, err := u.requirePaidPlan(req.Plan)
	if err != nil {
		return nil, err
	}

	sub, err := u.subRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if sub.PlanID == plan.ID && sub.Status == "active" {
		return &domain.BillingActionResponse{
			Subscription: sub,
			Message:      "plan already active",
		}, nil
	}

	if u.gateway == nil {
		return nil, domain.ErrBillingNotConfigured
	}

	if strings.TrimSpace(sub.ProviderSubscriptionID) == "" || sub.Status != "active" {
		return u.CreateCheckout(ctx, tenantID, domain.CreateCheckoutRequest{Plan: req.Plan})
	}

	updated, err := u.gateway.UpdateSubscriptionPlan(ctx, sub.ProviderSubscriptionID, u.gateway.PriceIDForPlan(plan.ID))
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sub.PlanID = plan.ID
	sub.Plan = &plan
	sub.Status = updated.Status
	sub.Provider = updated.Provider
	sub.ProviderCustomerID = fallback(updated.ProviderCustomerID, sub.ProviderCustomerID)
	sub.ProviderSubscriptionID = fallback(updated.ProviderSubscriptionID, sub.ProviderSubscriptionID)
	sub.ProviderPriceID = fallback(updated.ProviderPriceID, u.gateway.PriceIDForPlan(plan.ID))
	sub.CurrentPeriodStart = updated.CurrentPeriodStart
	if !updated.PeriodEnd.IsZero() {
		sub.PeriodEnd = updated.PeriodEnd
	}
	sub.TrialEndsAt = nil
	sub.CancelAtPeriodEnd = updated.CancelAtPeriodEnd
	sub.UpdatedAt = now
	if sub.ID == "" {
		sub.ID = uuid.NewString()
	}

	if err := u.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}

	return &domain.BillingActionResponse{
		Subscription: sub,
		Provider:     sub.Provider,
		Message:      "subscription updated",
	}, nil
}

func (u *BillingUsecase) CancelSubscription(ctx context.Context, tenantID string) (*domain.BillingActionResponse, error) {
	if u.gateway == nil {
		return nil, domain.ErrBillingNotConfigured
	}

	sub, err := u.subRepo.GetByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sub.ProviderSubscriptionID) == "" {
		return nil, fmt.Errorf("%w: no active paid subscription is linked", domain.ErrBadRequest)
	}

	updated, err := u.gateway.CancelSubscription(ctx, sub.ProviderSubscriptionID)
	if err != nil {
		return nil, err
	}
	sub.CancelAtPeriodEnd = updated.CancelAtPeriodEnd
	sub.Status = updated.Status
	if !updated.PeriodEnd.IsZero() {
		sub.PeriodEnd = updated.PeriodEnd
	}
	sub.UpdatedAt = time.Now().UTC()
	if err := u.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}

	return &domain.BillingActionResponse{
		Subscription: sub,
		Message:      "subscription will cancel at period end",
	}, nil
}

func (u *BillingUsecase) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	if u.gateway == nil {
		return domain.ErrBillingNotConfigured
	}

	event, err := u.gateway.ParseWebhook(payload, signature)
	if err != nil {
		return err
	}
	if event == nil {
		return nil
	}

	tenantID := strings.TrimSpace(event.TenantID)
	if tenantID == "" && strings.TrimSpace(event.ProviderSubscriptionID) != "" {
		current, lookupErr := u.subRepo.GetByProviderSubscriptionID(ctx, event.Provider, event.ProviderSubscriptionID)
		if lookupErr == nil {
			tenantID = current.TenantID
		}
	}
	if tenantID == "" {
		return fmt.Errorf("%w: tenant could not be resolved", domain.ErrBillingWebhookFailed)
	}

	current, err := u.subRepo.GetByTenant(ctx, tenantID)
	if err != nil && err != domain.ErrNoSubscription {
		return err
	}
	if current == nil {
		current = &domain.Subscription{
			ID:        uuid.NewString(),
			TenantID:  tenantID,
			CreatedAt: time.Now().UTC(),
		}
	}

	if event.PlanID == "" {
		event.PlanID = current.PlanID
	}
	if event.PlanID == "" {
		return fmt.Errorf("%w: plan could not be resolved", domain.ErrBillingWebhookFailed)
	}

	plan := planByID(event.PlanID)
	if plan == nil {
		return fmt.Errorf("%w: unknown plan id %s", domain.ErrBillingWebhookFailed, event.PlanID)
	}

	now := time.Now().UTC()
	current.PlanID = plan.ID
	current.Plan = plan
	current.Status = event.Status
	current.Provider = fallback(event.Provider, current.Provider)
	current.ProviderCustomerID = fallback(event.ProviderCustomerID, current.ProviderCustomerID)
	current.ProviderSubscriptionID = fallback(event.ProviderSubscriptionID, current.ProviderSubscriptionID)
	current.ProviderPriceID = fallback(event.ProviderPriceID, current.ProviderPriceID)
	current.CancelAtPeriodEnd = event.CancelAtPeriodEnd
	current.UpdatedAt = now
	if event.CurrentPeriodStart != nil {
		current.CurrentPeriodStart = event.CurrentPeriodStart
	}
	if event.PeriodEnd != nil {
		current.PeriodEnd = *event.PeriodEnd
	}
	if current.Status == "active" {
		current.TrialEndsAt = nil
	}
	if current.CreatedAt.IsZero() {
		current.CreatedAt = now
	}
	if current.ID == "" {
		current.ID = uuid.NewString()
	}

	if err := u.subRepo.Upsert(ctx, current); err != nil {
		return err
	}

	u.log.WithContext(ctx).Info("billing webhook applied", map[string]interface{}{
		"tenant_id":    tenantID,
		"event_type":   event.EventType,
		"plan_id":      current.PlanID,
		"status":       current.Status,
		"provider_sub": current.ProviderSubscriptionID,
	})
	return nil
}

func (u *BillingUsecase) requirePaidPlan(name domain.PlanName) (domain.Plan, error) {
	if name == "" || name == domain.PlanTrial {
		return domain.Plan{}, fmt.Errorf("%w: choose a paid plan", domain.ErrBadRequest)
	}
	plan, ok := domain.PlanByName(name)
	if !ok || plan.Name == domain.PlanTrial {
		return domain.Plan{}, fmt.Errorf("%w: invalid plan", domain.ErrBadRequest)
	}
	return plan, nil
}

func (u *BillingUsecase) appURL(path string) string {
	base := u.appBaseURL
	if base == "" {
		base = "http://localhost:3000"
	}
	return base + path
}

func fallback(value, current string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return current
}

func planByID(planID string) *domain.Plan {
	for _, plan := range domain.DefaultPlans() {
		if plan.ID == planID {
			copied := plan
			return &copied
		}
	}
	return nil
}
