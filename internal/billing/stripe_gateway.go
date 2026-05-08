package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v84"
	billingportalsession "github.com/stripe/stripe-go/v84/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v84/checkout/session"
	"github.com/stripe/stripe-go/v84/subscription"
	"github.com/whatsapp-saas/api/internal/domain"
)

const stripeProvider = "stripe"

type StripeGateway struct {
	webhookSecret string
	priceByPlan   map[string]string
	planByPrice   map[string]string
}

func NewStripeGateway(secretKey, webhookSecret string, priceByPlan map[string]string) *StripeGateway {
	if strings.TrimSpace(secretKey) == "" {
		return nil
	}
	stripe.Key = secretKey

	normalizedPrices := make(map[string]string, len(priceByPlan))
	reverse := make(map[string]string, len(priceByPlan))
	for planID, priceID := range priceByPlan {
		priceID = strings.TrimSpace(priceID)
		normalizedPrices[planID] = priceID
		if priceID != "" {
			reverse[priceID] = planID
		}
	}

	return &StripeGateway{
		webhookSecret: strings.TrimSpace(webhookSecret),
		priceByPlan:   normalizedPrices,
		planByPrice:   reverse,
	}
}

func (g *StripeGateway) PriceIDForPlan(planID string) string {
	if g == nil {
		return ""
	}
	return g.priceByPlan[planID]
}

func (g *StripeGateway) CreateCheckoutSession(ctx context.Context, req domain.CheckoutSessionRequest) (*domain.CheckoutSessionResponse, error) {
	if g == nil {
		return nil, domain.ErrBillingNotConfigured
	}
	priceID := g.PriceIDForPlan(req.Plan.ID)
	if priceID == "" {
		return nil, domain.ErrBillingNotConfigured
	}

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:        stripe.String(req.SuccessURL),
		CancelURL:         stripe.String(req.CancelURL),
		ClientReferenceID: stripe.String(req.TenantID),
		CustomerEmail:     stripe.String(req.CustomerEmail),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: map[string]string{
			"tenant_id": req.TenantID,
			"tenant":    req.TenantName,
			"plan_id":   req.Plan.ID,
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"tenant_id": req.TenantID,
				"plan_id":   req.Plan.ID,
			},
		},
	}

	session, err := checkoutsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe checkout session: %w", err)
	}

	return &domain.CheckoutSessionResponse{
		URL:       session.URL,
		SessionID: session.ID,
		Provider:  stripeProvider,
	}, nil
}

func (g *StripeGateway) CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (*domain.BillingPortalResponse, error) {
	if g == nil {
		return nil, domain.ErrBillingNotConfigured
	}
	if strings.TrimSpace(customerID) == "" {
		return nil, fmt.Errorf("%w: customer id is required", domain.ErrBadRequest)
	}

	session, err := billingportalsession.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	})
	if err != nil {
		return nil, fmt.Errorf("stripe billing portal session: %w", err)
	}

	return &domain.BillingPortalResponse{
		URL:      session.URL,
		Provider: stripeProvider,
	}, nil
}

func (g *StripeGateway) UpdateSubscriptionPlan(ctx context.Context, subscriptionID, priceID string) (*domain.Subscription, error) {
	if g == nil {
		return nil, domain.ErrBillingNotConfigured
	}
	if strings.TrimSpace(subscriptionID) == "" || strings.TrimSpace(priceID) == "" {
		return nil, fmt.Errorf("%w: subscription id and price id are required", domain.ErrBadRequest)
	}

	current, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get subscription: %w", err)
	}
	if current.Items == nil || len(current.Items.Data) == 0 {
		return nil, fmt.Errorf("%w: subscription has no billable items", domain.ErrBadRequest)
	}

	updated, err := subscription.Update(subscriptionID, &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(current.Items.Data[0].ID),
				Price: stripe.String(priceID),
			},
		},
		ProrationBehavior: stripe.String("create_prorations"),
	})
	if err != nil {
		return nil, fmt.Errorf("stripe update subscription: %w", err)
	}
	return g.domainSubscriptionFromStripe(updated), nil
}

func (g *StripeGateway) CancelSubscription(ctx context.Context, subscriptionID string) (*domain.Subscription, error) {
	if g == nil {
		return nil, domain.ErrBillingNotConfigured
	}
	if strings.TrimSpace(subscriptionID) == "" {
		return nil, fmt.Errorf("%w: subscription id is required", domain.ErrBadRequest)
	}

	updated, err := subscription.Update(subscriptionID, &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("stripe cancel subscription: %w", err)
	}
	return g.domainSubscriptionFromStripe(updated), nil
}

func (g *StripeGateway) ParseWebhook(payload []byte, signature string) (*domain.BillingWebhookEvent, error) {
	if g == nil || g.webhookSecret == "" {
		return nil, domain.ErrBillingNotConfigured
	}

	event, err := stripe.ConstructEvent(payload, signature, g.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrBillingWebhookFailed, err)
	}

	switch string(event.Type) {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return nil, fmt.Errorf("%w: decode checkout session", domain.ErrBillingWebhookFailed)
		}
		result := &domain.BillingWebhookEvent{
			EventType:         string(event.Type),
			TenantID:          strings.TrimSpace(session.ClientReferenceID),
			PlanID:            strings.TrimSpace(session.Metadata["plan_id"]),
			Status:            "active",
			Provider:          stripeProvider,
			ProviderPriceID:   g.PriceIDForPlan(strings.TrimSpace(session.Metadata["plan_id"])),
			CancelAtPeriodEnd: false,
		}
		if session.Customer != nil {
			result.ProviderCustomerID = session.Customer.ID
		}
		if session.Subscription != nil {
			result.ProviderSubscriptionID = session.Subscription.ID
		}
		if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid &&
			session.PaymentStatus != stripe.CheckoutSessionPaymentStatusNoPaymentRequired {
			result.Status = "pending"
		}
		return result, nil
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return nil, fmt.Errorf("%w: decode subscription", domain.ErrBillingWebhookFailed)
		}
		return g.webhookEventFromStripeSubscription(string(event.Type), &sub), nil
	default:
		return nil, nil
	}
}

func (g *StripeGateway) webhookEventFromStripeSubscription(eventType string, sub *stripe.Subscription) *domain.BillingWebhookEvent {
	var planID string
	var priceID string
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.Price != nil {
			priceID = item.Price.ID
			planID = g.planByPrice[priceID]
		}
	}

	result := &domain.BillingWebhookEvent{
		EventType:              eventType,
		TenantID:               strings.TrimSpace(sub.Metadata["tenant_id"]),
		PlanID:                 planID,
		Status:                 mapStripeSubscriptionStatus(string(sub.Status)),
		Provider:               stripeProvider,
		ProviderPriceID:        priceID,
		ProviderSubscriptionID: sub.ID,
		CancelAtPeriodEnd:      sub.CancelAtPeriodEnd,
	}
	if sub.Customer != nil {
		result.ProviderCustomerID = sub.Customer.ID
	}
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]
		if item.CurrentPeriodStart > 0 {
			start := time.Unix(item.CurrentPeriodStart, 0).UTC()
			result.CurrentPeriodStart = &start
		}
		if item.CurrentPeriodEnd > 0 {
			end := time.Unix(item.CurrentPeriodEnd, 0).UTC()
			result.PeriodEnd = &end
		}
	}
	if result.PlanID == "" {
		result.PlanID = strings.TrimSpace(sub.Metadata["plan_id"])
	}
	return result
}

func (g *StripeGateway) domainSubscriptionFromStripe(sub *stripe.Subscription) *domain.Subscription {
	event := g.webhookEventFromStripeSubscription("manual.sync", sub)
	now := time.Now().UTC()
	periodEnd := now
	if event.PeriodEnd != nil {
		periodEnd = *event.PeriodEnd
	}
	return &domain.Subscription{
		PlanID:                 event.PlanID,
		Status:                 event.Status,
		Provider:               event.Provider,
		ProviderCustomerID:     event.ProviderCustomerID,
		ProviderSubscriptionID: event.ProviderSubscriptionID,
		ProviderPriceID:        event.ProviderPriceID,
		CurrentPeriodStart:     event.CurrentPeriodStart,
		PeriodEnd:              periodEnd,
		CancelAtPeriodEnd:      event.CancelAtPeriodEnd,
		UpdatedAt:              now,
	}
}

func mapStripeSubscriptionStatus(status string) string {
	switch status {
	case "active", "trialing":
		return "active"
	case "past_due", "unpaid", "paused":
		return "past_due"
	case "canceled", "incomplete_expired":
		return "cancelled"
	case "incomplete":
		return "pending"
	default:
		return "pending"
	}
}
