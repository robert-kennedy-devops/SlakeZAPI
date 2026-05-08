package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
)

func SyncDefaultPlans(ctx context.Context, db *sql.DB) error {
	for _, plan := range domain.DefaultPlans() {
		_, err := db.ExecContext(ctx, `
			INSERT INTO plans (id, name, monthly_limit, price_usd_cents, webhook_enabled)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE
			  SET name = EXCLUDED.name,
			      monthly_limit = EXCLUDED.monthly_limit,
			      price_usd_cents = EXCLUDED.price_usd_cents,
			      webhook_enabled = EXCLUDED.webhook_enabled
		`, plan.ID, plan.Name, plan.MonthlyLimit, plan.PriceUSDCents, plan.WebhookEnabled)
		if err != nil {
			return fmt.Errorf("sync default plans: %w", err)
		}
	}
	return nil
}

func EnsureSubscriptionSchema(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS provider_customer_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS provider_subscription_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS provider_price_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS current_period_start TIMESTAMPTZ`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS cancel_at_period_end BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_provider_subscription_id ON subscriptions (provider_subscription_id)`,
	}
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("ensure subscription schema: %w", err)
		}
	}
	return nil
}

// ─── Webhook Repository ──────────────────────────────────────────────────────

type webhookRepo struct{ db *sql.DB }

func NewWebhookRepository(db *sql.DB) domain.WebhookRepository {
	return &webhookRepo{db: db}
}

func (r *webhookRepo) Create(ctx context.Context, wh *domain.Webhook) error {
	q := `
		INSERT INTO webhooks (id, tenant_id, instance_id, url, events, secret, active, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	events := strings.Join(wh.Events, ",")
	_, err := r.db.ExecContext(ctx, q,
		wh.ID, wh.TenantID, nullableString(wh.InstanceID), wh.URL, events, wh.Secret, wh.Active, wh.CreatedAt)
	if err != nil {
		return fmt.Errorf("webhookRepo.Create: %w", err)
	}
	return nil
}

func (r *webhookRepo) GetByTenant(ctx context.Context, tenantID, instanceID string) ([]domain.Webhook, error) {
	q := `SELECT id, tenant_id, instance_id, url, events, secret, active, created_at
	      FROM webhooks WHERE tenant_id = $1 AND active = true AND ($2 = '' OR COALESCE(instance_id, '') = $2)`
	rows, err := r.db.QueryContext(ctx, q, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("webhookRepo.GetByTenant: %w", err)
	}
	defer rows.Close()

	var out []domain.Webhook
	for rows.Next() {
		wh := domain.Webhook{}
		var events string
		var dbInstanceID sql.NullString
		if err := rows.Scan(&wh.ID, &wh.TenantID, &dbInstanceID, &wh.URL, &events, &wh.Secret, &wh.Active, &wh.CreatedAt); err != nil {
			return nil, err
		}
		wh.InstanceID = dbInstanceID.String
		wh.Events = strings.Split(events, ",")
		out = append(out, wh)
	}
	return out, rows.Err()
}

func (r *webhookRepo) GetByID(ctx context.Context, id string) (*domain.Webhook, error) {
	q := `SELECT id, tenant_id, instance_id, url, events, secret, active, created_at FROM webhooks WHERE id = $1`
	wh := &domain.Webhook{}
	var events string
	var dbInstanceID sql.NullString
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&wh.ID, &wh.TenantID, &dbInstanceID, &wh.URL, &events, &wh.Secret, &wh.Active, &wh.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrWebhookNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("webhookRepo.GetByID: %w", err)
	}
	wh.InstanceID = dbInstanceID.String
	wh.Events = strings.Split(events, ",")
	return wh, nil
}

func (r *webhookRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE webhooks SET active=false WHERE id=$1`, id)
	return err
}

func (r *webhookRepo) CreateDelivery(ctx context.Context, delivery *domain.WebhookDelivery) error {
	q := `
		INSERT INTO webhook_deliveries (
			id, webhook_id, tenant_id, instance_id, event_type, webhook_url, status,
			attempts, response_status, response_body, last_error, payload_json,
			delivered_at, last_attempt_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`
	_, err := r.db.ExecContext(ctx, q,
		delivery.ID,
		delivery.WebhookID,
		delivery.TenantID,
		nullableString(delivery.InstanceID),
		string(delivery.EventType),
		delivery.WebhookURL,
		string(delivery.Status),
		delivery.Attempts,
		delivery.ResponseStatus,
		delivery.ResponseBody,
		delivery.LastError,
		delivery.PayloadJSON,
		delivery.DeliveredAt,
		delivery.LastAttemptAt,
		delivery.CreatedAt,
		delivery.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("webhookRepo.CreateDelivery: %w", err)
	}
	return nil
}

func (r *webhookRepo) UpdateDeliveryAttempt(ctx context.Context, id string, status domain.WebhookDeliveryStatus, responseStatus int, responseBody, lastError string, deliveredAt, attemptedAt *time.Time) error {
	q := `
		UPDATE webhook_deliveries
		SET status = $2,
			attempts = attempts + 1,
			response_status = $3,
			response_body = $4,
			last_error = $5,
			delivered_at = COALESCE($6, delivered_at),
			last_attempt_at = $7,
			updated_at = NOW()
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, q, id, string(status), responseStatus, responseBody, lastError, deliveredAt, attemptedAt)
	if err != nil {
		return fmt.Errorf("webhookRepo.UpdateDeliveryAttempt: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return domain.ErrWebhookDeliveryNotFound
	}
	return err
}

func (r *webhookRepo) ListDeliveries(ctx context.Context, tenantID, instanceID, webhookID string, limit int) ([]domain.WebhookDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `
		SELECT id, webhook_id, tenant_id, instance_id, event_type, webhook_url, status,
		       attempts, response_status, response_body, last_error, payload_json,
		       delivered_at, last_attempt_at, created_at, updated_at
		FROM webhook_deliveries
		WHERE tenant_id = $1
		  AND ($2 = '' OR COALESCE(instance_id, '') = $2)
		  AND ($3 = '' OR webhook_id = $3)
		ORDER BY created_at DESC
		LIMIT $4
	`
	rows, err := r.db.QueryContext(ctx, q, tenantID, instanceID, webhookID, limit)
	if err != nil {
		return nil, fmt.Errorf("webhookRepo.ListDeliveries: %w", err)
	}
	defer rows.Close()

	out := make([]domain.WebhookDelivery, 0, limit)
	for rows.Next() {
		var item domain.WebhookDelivery
		var instanceID sql.NullString
		var eventType string
		var status string
		if err := rows.Scan(
			&item.ID,
			&item.WebhookID,
			&item.TenantID,
			&instanceID,
			&eventType,
			&item.WebhookURL,
			&status,
			&item.Attempts,
			&item.ResponseStatus,
			&item.ResponseBody,
			&item.LastError,
			&item.PayloadJSON,
			&item.DeliveredAt,
			&item.LastAttemptAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("webhookRepo.ListDeliveries.scan: %w", err)
		}
		item.InstanceID = instanceID.String
		item.EventType = domain.EventType(eventType)
		item.Status = domain.WebhookDeliveryStatus(status)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *webhookRepo) GetDeliveryByID(ctx context.Context, id string) (*domain.WebhookDelivery, error) {
	q := `
		SELECT id, webhook_id, tenant_id, instance_id, event_type, webhook_url, status,
		       attempts, response_status, response_body, last_error, payload_json,
		       delivered_at, last_attempt_at, created_at, updated_at
		FROM webhook_deliveries
		WHERE id = $1
	`
	var item domain.WebhookDelivery
	var instanceID sql.NullString
	var eventType string
	var status string
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&item.ID,
		&item.WebhookID,
		&item.TenantID,
		&instanceID,
		&eventType,
		&item.WebhookURL,
		&status,
		&item.Attempts,
		&item.ResponseStatus,
		&item.ResponseBody,
		&item.LastError,
		&item.PayloadJSON,
		&item.DeliveredAt,
		&item.LastAttemptAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrWebhookDeliveryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("webhookRepo.GetDeliveryByID: %w", err)
	}
	item.InstanceID = instanceID.String
	item.EventType = domain.EventType(eventType)
	item.Status = domain.WebhookDeliveryStatus(status)
	return &item, nil
}

// ─── Subscription Repository ─────────────────────────────────────────────────

type subscriptionRepo struct{ db *sql.DB }

func NewSubscriptionRepository(db *sql.DB) domain.SubscriptionRepository {
	return &subscriptionRepo{db: db}
}

func (r *subscriptionRepo) GetByTenant(ctx context.Context, tenantID string) (*domain.Subscription, error) {
	q := `
		SELECT s.id, s.tenant_id, s.plan_id, s.status, s.provider, s.provider_customer_id,
		       s.provider_subscription_id, s.provider_price_id, s.current_period_start,
		       s.period_end, s.trial_ends_at, s.cancel_at_period_end, s.created_at, s.updated_at,
		       p.id, p.name, p.monthly_limit, p.price_usd_cents, p.webhook_enabled
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.tenant_id = $1
		  AND s.status IN ('trial', 'pending', 'active', 'past_due', 'cancelled')
		ORDER BY s.updated_at DESC, s.created_at DESC LIMIT 1
	`
	return r.scanSubscription(ctx, q, tenantID)
}

func (r *subscriptionRepo) GetByProviderSubscriptionID(ctx context.Context, provider, subscriptionID string) (*domain.Subscription, error) {
	q := `
		SELECT s.id, s.tenant_id, s.plan_id, s.status, s.provider, s.provider_customer_id,
		       s.provider_subscription_id, s.provider_price_id, s.current_period_start,
		       s.period_end, s.trial_ends_at, s.cancel_at_period_end, s.created_at, s.updated_at,
		       p.id, p.name, p.monthly_limit, p.price_usd_cents, p.webhook_enabled
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.provider = $1 AND s.provider_subscription_id = $2
		LIMIT 1
	`
	return r.scanSubscription(ctx, q, provider, subscriptionID)
}

func (r *subscriptionRepo) scanSubscription(ctx context.Context, query string, args ...interface{}) (*domain.Subscription, error) {
	sub := &domain.Subscription{Plan: &domain.Plan{}}
	var currentPeriodStart sql.NullTime
	var trialEndsAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Status, &sub.Provider, &sub.ProviderCustomerID,
		&sub.ProviderSubscriptionID, &sub.ProviderPriceID, &currentPeriodStart,
		&sub.PeriodEnd, &trialEndsAt, &sub.CancelAtPeriodEnd, &sub.CreatedAt, &sub.UpdatedAt,
		&sub.Plan.ID, &sub.Plan.Name, &sub.Plan.MonthlyLimit, &sub.Plan.PriceUSDCents, &sub.Plan.WebhookEnabled,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNoSubscription
	}
	if err != nil {
		return nil, fmt.Errorf("subscriptionRepo.scanSubscription: %w", err)
	}
	if currentPeriodStart.Valid {
		t := currentPeriodStart.Time
		sub.CurrentPeriodStart = &t
	}
	if trialEndsAt.Valid {
		t := trialEndsAt.Time
		sub.TrialEndsAt = &t
	}
	return sub, nil
}

func (r *subscriptionRepo) Upsert(ctx context.Context, sub *domain.Subscription) error {
	q := `
		INSERT INTO subscriptions (
			id, tenant_id, plan_id, status, provider, provider_customer_id,
			provider_subscription_id, provider_price_id, current_period_start,
			period_end, trial_ends_at, cancel_at_period_end, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (tenant_id) DO UPDATE
		  SET plan_id=$3,
		      status=$4,
		      provider=$5,
		      provider_customer_id=$6,
		      provider_subscription_id=$7,
		      provider_price_id=$8,
		      current_period_start=$9,
		      period_end=$10,
		      trial_ends_at=$11,
		      cancel_at_period_end=$12,
		      updated_at=$14
	`
	_, err := r.db.ExecContext(ctx, q,
		sub.ID, sub.TenantID, sub.PlanID, sub.Status, sub.Provider, sub.ProviderCustomerID,
		sub.ProviderSubscriptionID, sub.ProviderPriceID, sub.CurrentPeriodStart,
		sub.PeriodEnd, sub.TrialEndsAt, sub.CancelAtPeriodEnd, sub.CreatedAt, sub.UpdatedAt)
	return err
}

// ─── Usage Repository ────────────────────────────────────────────────────────

type usageRepo struct{ db *sql.DB }

func NewUsageRepository(db *sql.DB) domain.UsageRepository {
	return &usageRepo{db: db}
}

func currentMonth() string {
	return time.Now().UTC().Format("2006-01")
}

func (r *usageRepo) IncrementSent(ctx context.Context, tenantID string) error {
	q := `
		INSERT INTO usage (tenant_id, month, sent, received, updated_at)
		VALUES ($1, $2, 1, 0, NOW())
		ON CONFLICT (tenant_id, month) DO UPDATE
		  SET sent = usage.sent + 1, updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, q, tenantID, currentMonth())
	return err
}

func (r *usageRepo) IncrementReceived(ctx context.Context, tenantID string) error {
	q := `
		INSERT INTO usage (tenant_id, month, sent, received, updated_at)
		VALUES ($1, $2, 0, 1, NOW())
		ON CONFLICT (tenant_id, month) DO UPDATE
		  SET received = usage.received + 1, updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, q, tenantID, currentMonth())
	return err
}

func (r *usageRepo) GetCurrentMonth(ctx context.Context, tenantID string) (*domain.Usage, error) {
	q := `SELECT tenant_id, month, sent, received, updated_at
	      FROM usage WHERE tenant_id=$1 AND month=$2`
	u := &domain.Usage{}
	err := r.db.QueryRowContext(ctx, q, tenantID, currentMonth()).Scan(
		&u.TenantID, &u.Month, &u.Sent, &u.Received, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return &domain.Usage{TenantID: tenantID, Month: currentMonth()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("usageRepo.GetCurrentMonth: %w", err)
	}
	return u, nil
}
