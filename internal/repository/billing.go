package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
)

// ─── Webhook Repository ──────────────────────────────────────────────────────

type webhookRepo struct{ db *sql.DB }

func NewWebhookRepository(db *sql.DB) domain.WebhookRepository {
	return &webhookRepo{db: db}
}

func (r *webhookRepo) Create(ctx context.Context, wh *domain.Webhook) error {
	q := `
		INSERT INTO webhooks (id, tenant_id, url, events, secret, active, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`
	events := strings.Join(wh.Events, ",")
	_, err := r.db.ExecContext(ctx, q,
		wh.ID, wh.TenantID, wh.URL, events, wh.Secret, wh.Active, wh.CreatedAt)
	if err != nil {
		return fmt.Errorf("webhookRepo.Create: %w", err)
	}
	return nil
}

func (r *webhookRepo) GetByTenant(ctx context.Context, tenantID string) ([]domain.Webhook, error) {
	q := `SELECT id, tenant_id, url, events, secret, active, created_at
	      FROM webhooks WHERE tenant_id = $1 AND active = true`
	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("webhookRepo.GetByTenant: %w", err)
	}
	defer rows.Close()

	var out []domain.Webhook
	for rows.Next() {
		wh := domain.Webhook{}
		var events string
		if err := rows.Scan(&wh.ID, &wh.TenantID, &wh.URL, &events, &wh.Secret, &wh.Active, &wh.CreatedAt); err != nil {
			return nil, err
		}
		wh.Events = strings.Split(events, ",")
		out = append(out, wh)
	}
	return out, rows.Err()
}

func (r *webhookRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE webhooks SET active=false WHERE id=$1`, id)
	return err
}

// ─── Subscription Repository ─────────────────────────────────────────────────

type subscriptionRepo struct{ db *sql.DB }

func NewSubscriptionRepository(db *sql.DB) domain.SubscriptionRepository {
	return &subscriptionRepo{db: db}
}

func (r *subscriptionRepo) GetByTenant(ctx context.Context, tenantID string) (*domain.Subscription, error) {
	q := `
		SELECT s.id, s.tenant_id, s.plan_id, s.status, s.period_end, s.created_at,
		       p.id, p.name, p.monthly_limit, p.price_usd_cents, p.webhook_enabled
		FROM subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE s.tenant_id = $1 AND s.status = 'active'
		ORDER BY s.created_at DESC LIMIT 1
	`
	sub := &domain.Subscription{Plan: &domain.Plan{}}
	err := r.db.QueryRowContext(ctx, q, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &sub.Status, &sub.PeriodEnd, &sub.CreatedAt,
		&sub.Plan.ID, &sub.Plan.Name, &sub.Plan.MonthlyLimit, &sub.Plan.PriceUSDCents, &sub.Plan.WebhookEnabled,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNoSubscription
	}
	if err != nil {
		return nil, fmt.Errorf("subscriptionRepo.GetByTenant: %w", err)
	}
	return sub, nil
}

func (r *subscriptionRepo) Upsert(ctx context.Context, sub *domain.Subscription) error {
	q := `
		INSERT INTO subscriptions (id, tenant_id, plan_id, status, period_end, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (tenant_id) DO UPDATE
		  SET plan_id=$3, status=$4, period_end=$5
	`
	_, err := r.db.ExecContext(ctx, q,
		sub.ID, sub.TenantID, sub.PlanID, sub.Status, sub.PeriodEnd, sub.CreatedAt)
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
