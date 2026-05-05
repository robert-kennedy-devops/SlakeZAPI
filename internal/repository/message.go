package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/whatsapp-saas/api/internal/domain"
)

type messageRepo struct {
	db *sql.DB
}

// NewMessageRepository returns a PostgreSQL-backed MessageRepository.
func NewMessageRepository(db *sql.DB) domain.MessageRepository {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, m *domain.Message) error {
	q := `
		INSERT INTO messages
			(id, tenant_id, whatsapp_id, phone, body, type, mime_type, file_name, media_url, direct_path, file_length, media_key, file_sha256, file_enc_sha256, direction, status, sent_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`
	_, err := r.db.ExecContext(ctx, q,
		m.ID, m.TenantID, m.WhatsAppID, m.Phone,
		m.Body, m.Type, m.MimeType, m.FileName, m.MediaURL, m.DirectPath, m.FileLength,
		m.MediaKey, m.FileSHA256, m.FileEncSHA256, m.Direction, string(m.Status), m.SentAt, m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("messageRepo.Create: %w", err)
	}
	return nil
}

func (r *messageRepo) GetByID(ctx context.Context, id string) (*domain.Message, error) {
	q := `
		SELECT id, tenant_id, whatsapp_id, phone, body, type, mime_type, file_name, media_url, direct_path, file_length, media_key, file_sha256, file_enc_sha256, direction, status, sent_at, created_at
		FROM messages WHERE id = $1
	`
	m := &domain.Message{}
	var status string
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&m.ID, &m.TenantID, &m.WhatsAppID, &m.Phone,
		&m.Body, &m.Type, &m.MimeType, &m.FileName, &m.MediaURL, &m.DirectPath, &m.FileLength,
		&m.MediaKey, &m.FileSHA256, &m.FileEncSHA256, &m.Direction, &status, &m.SentAt, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrMessageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("messageRepo.GetByID: %w", err)
	}
	m.Status = domain.MessageStatus(status)
	return m, nil
}

func (r *messageRepo) GetByWhatsAppID(ctx context.Context, tenantID, whatsappID string) (*domain.Message, error) {
	q := `
		SELECT id, tenant_id, whatsapp_id, phone, body, type, mime_type, file_name, media_url, direct_path, file_length, media_key, file_sha256, file_enc_sha256, direction, status, sent_at, created_at
		FROM messages WHERE tenant_id = $1 AND whatsapp_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`
	m := &domain.Message{}
	var status string
	err := r.db.QueryRowContext(ctx, q, tenantID, whatsappID).Scan(
		&m.ID, &m.TenantID, &m.WhatsAppID, &m.Phone,
		&m.Body, &m.Type, &m.MimeType, &m.FileName, &m.MediaURL, &m.DirectPath, &m.FileLength,
		&m.MediaKey, &m.FileSHA256, &m.FileEncSHA256, &m.Direction, &status, &m.SentAt, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrMessageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("messageRepo.GetByWhatsAppID: %w", err)
	}
	m.Status = domain.MessageStatus(status)
	return m, nil
}

func (r *messageRepo) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]domain.Message, error) {
	q := `
		SELECT id, tenant_id, whatsapp_id, phone, body, type, mime_type, file_name, media_url, direct_path, file_length, media_key, file_sha256, file_enc_sha256, direction, status, sent_at, created_at
		FROM messages WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, q, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("messageRepo.ListByTenant: %w", err)
	}
	defer rows.Close()

	var msgs []domain.Message
	for rows.Next() {
		m := domain.Message{}
		var status string
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.WhatsAppID, &m.Phone,
			&m.Body, &m.Type, &m.MimeType, &m.FileName, &m.MediaURL, &m.DirectPath, &m.FileLength,
			&m.MediaKey, &m.FileSHA256, &m.FileEncSHA256, &m.Direction, &status, &m.SentAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.Status = domain.MessageStatus(status)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *messageRepo) UpdateStatus(ctx context.Context, id string, status domain.MessageStatus) error {
	q := `UPDATE messages SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, string(status), id)
	if err != nil {
		return fmt.Errorf("messageRepo.UpdateStatus: %w", err)
	}
	return nil
}
