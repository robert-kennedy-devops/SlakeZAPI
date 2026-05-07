package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
)

type campaignRepo struct {
	db *sql.DB
}

func NewCampaignRepository(db *sql.DB) domain.CampaignRepository {
	return &campaignRepo{db: db}
}

func (r *campaignRepo) Create(ctx context.Context, campaign *domain.Campaign, recipients []domain.CampaignRecipient) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("campaignRepo.Create begin: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO campaigns (id, tenant_id, instance_id, name, message, status, scheduled_at, last_executed_at, total_contacts, sent_count, failed_count, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, campaign.ID, campaign.TenantID, campaign.InstanceID, campaign.Name, campaign.Message, string(campaign.Status), campaign.ScheduledAt, campaign.LastExecutedAt, campaign.TotalContacts, campaign.SentCount, campaign.FailedCount, campaign.CreatedAt, campaign.UpdatedAt)
	if err != nil {
		return fmt.Errorf("campaignRepo.Create campaign: %w", err)
	}

	for _, recipient := range recipients {
		var variables map[string]string
		if recipient.Variables != "" {
			if err := json.Unmarshal([]byte(recipient.Variables), &variables); err != nil {
				return fmt.Errorf("campaignRepo.Create recipient variables: %w", err)
			}
		}
		payload, _ := json.Marshal(variables)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO campaign_recipients (id, campaign_id, input_phone, phone, name, variables, message_id, status, error, is_whatsapp, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		`, recipient.ID, recipient.CampaignID, recipient.InputPhone, recipient.Phone, recipient.Name, payload, recipient.MessageID, string(recipient.Status), recipient.Error, recipient.IsWhatsApp, recipient.CreatedAt, recipient.UpdatedAt); err != nil {
			return fmt.Errorf("campaignRepo.Create recipient: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("campaignRepo.Create commit: %w", err)
	}
	return nil
}

func (r *campaignRepo) GetByID(ctx context.Context, id string) (*domain.Campaign, error) {
	var campaign domain.Campaign
	var status string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, instance_id, name, message, status, scheduled_at, last_executed_at, total_contacts, sent_count, failed_count, created_at, updated_at
		FROM campaigns WHERE id = $1
	`, id).Scan(&campaign.ID, &campaign.TenantID, &campaign.InstanceID, &campaign.Name, &campaign.Message, &status, &campaign.ScheduledAt, &campaign.LastExecutedAt, &campaign.TotalContacts, &campaign.SentCount, &campaign.FailedCount, &campaign.CreatedAt, &campaign.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrCampaignNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("campaignRepo.GetByID: %w", err)
	}
	campaign.Status = domain.CampaignStatus(status)
	return &campaign, nil
}

func (r *campaignRepo) ListByTenant(ctx context.Context, tenantID, instanceID string) ([]domain.Campaign, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, instance_id, name, message, status, scheduled_at, last_executed_at, total_contacts, sent_count, failed_count, created_at, updated_at
		FROM campaigns
		WHERE tenant_id = $1 AND ($2 = '' OR instance_id = $2)
		ORDER BY created_at DESC
	`, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("campaignRepo.ListByTenant: %w", err)
	}
	defer rows.Close()

	var campaigns []domain.Campaign
	for rows.Next() {
		var campaign domain.Campaign
		var status string
		if err := rows.Scan(&campaign.ID, &campaign.TenantID, &campaign.InstanceID, &campaign.Name, &campaign.Message, &status, &campaign.ScheduledAt, &campaign.LastExecutedAt, &campaign.TotalContacts, &campaign.SentCount, &campaign.FailedCount, &campaign.CreatedAt, &campaign.UpdatedAt); err != nil {
			return nil, err
		}
		campaign.Status = domain.CampaignStatus(status)
		campaigns = append(campaigns, campaign)
	}
	return campaigns, rows.Err()
}

func (r *campaignRepo) ListRecipients(ctx context.Context, campaignID string) ([]domain.CampaignRecipient, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, campaign_id, input_phone, phone, name, variables::text, message_id, status, error, is_whatsapp, created_at, updated_at
		FROM campaign_recipients
		WHERE campaign_id = $1
		ORDER BY created_at ASC
	`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("campaignRepo.ListRecipients: %w", err)
	}
	defer rows.Close()

	var recipients []domain.CampaignRecipient
	for rows.Next() {
		var recipient domain.CampaignRecipient
		var status string
		if err := rows.Scan(&recipient.ID, &recipient.CampaignID, &recipient.InputPhone, &recipient.Phone, &recipient.Name, &recipient.Variables, &recipient.MessageID, &status, &recipient.Error, &recipient.IsWhatsApp, &recipient.CreatedAt, &recipient.UpdatedAt); err != nil {
			return nil, err
		}
		recipient.Status = domain.MessageStatus(status)
		recipients = append(recipients, recipient)
	}
	return recipients, rows.Err()
}

func (r *campaignRepo) ListDue(ctx context.Context, now time.Time) ([]domain.Campaign, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, instance_id, name, message, status, scheduled_at, last_executed_at, total_contacts, sent_count, failed_count, created_at, updated_at
		FROM campaigns
		WHERE status = $1 AND scheduled_at IS NOT NULL AND scheduled_at <= $2
		ORDER BY scheduled_at ASC
	`, string(domain.CampaignStatusScheduled), now)
	if err != nil {
		return nil, fmt.Errorf("campaignRepo.ListDue: %w", err)
	}
	defer rows.Close()

	var campaigns []domain.Campaign
	for rows.Next() {
		var campaign domain.Campaign
		var status string
		if err := rows.Scan(&campaign.ID, &campaign.TenantID, &campaign.InstanceID, &campaign.Name, &campaign.Message, &status, &campaign.ScheduledAt, &campaign.LastExecutedAt, &campaign.TotalContacts, &campaign.SentCount, &campaign.FailedCount, &campaign.CreatedAt, &campaign.UpdatedAt); err != nil {
			return nil, err
		}
		campaign.Status = domain.CampaignStatus(status)
		campaigns = append(campaigns, campaign)
	}
	return campaigns, rows.Err()
}

func (r *campaignRepo) UpdateStatus(ctx context.Context, id string, status domain.CampaignStatus, lastExecutedAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE campaigns SET status = $1, last_executed_at = COALESCE($2, last_executed_at), updated_at = NOW() WHERE id = $3`, string(status), lastExecutedAt, id)
	if err != nil {
		return fmt.Errorf("campaignRepo.UpdateStatus: %w", err)
	}
	return nil
}

func (r *campaignRepo) UpdateRecipientResult(ctx context.Context, recipientID, phone, messageID string, status domain.MessageStatus, isWhatsApp bool, errText string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE campaign_recipients
		SET phone = COALESCE(NULLIF($1, ''), phone),
		    message_id = COALESCE(NULLIF($2, ''), message_id),
		    status = $3,
		    is_whatsapp = $4,
		    error = $5,
		    updated_at = NOW()
		WHERE id = $6
	`, phone, messageID, string(status), isWhatsApp, errText, recipientID)
	if err != nil {
		return fmt.Errorf("campaignRepo.UpdateRecipientResult: %w", err)
	}
	return nil
}

func (r *campaignRepo) UpdateCounters(ctx context.Context, campaignID string, sentCount, failedCount int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE campaigns SET sent_count = $1, failed_count = $2, updated_at = NOW() WHERE id = $3`, sentCount, failedCount, campaignID)
	if err != nil {
		return fmt.Errorf("campaignRepo.UpdateCounters: %w", err)
	}
	return nil
}

var _ domain.CampaignRepository = (*campaignRepo)(nil)
