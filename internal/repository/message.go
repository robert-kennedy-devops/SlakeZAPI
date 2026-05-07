package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
			(id, tenant_id, instance_id, whatsapp_id, phone, body, type, mime_type, file_name, media_url, direct_path, file_length, media_key, file_sha256, file_enc_sha256, direction, status, sent_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
	`
	_, err := r.db.ExecContext(ctx, q,
		m.ID, m.TenantID, nullableString(m.InstanceID), m.WhatsAppID, m.Phone,
		m.Body, m.Type, m.MimeType, m.FileName, m.MediaURL, m.DirectPath, m.FileLength,
		m.MediaKey, m.FileSHA256, m.FileEncSHA256, m.Direction, string(m.Status), m.SentAt, m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("messageRepo.Create: %w", err)
	}
	if err := r.upsertConversation(ctx, m); err != nil {
		return fmt.Errorf("messageRepo.Create conversation: %w", err)
	}
	return nil
}

func (r *messageRepo) GetByID(ctx context.Context, id string) (*domain.Message, error) {
	q := `
		SELECT id, tenant_id, instance_id, whatsapp_id, phone, body, type, mime_type, file_name, media_url, direct_path, file_length, media_key, file_sha256, file_enc_sha256, direction, status, sent_at, created_at
		FROM messages WHERE id = $1
	`
	m := &domain.Message{}
	var status string
	var instanceID sql.NullString
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&m.ID, &m.TenantID, &instanceID, &m.WhatsAppID, &m.Phone,
		&m.Body, &m.Type, &m.MimeType, &m.FileName, &m.MediaURL, &m.DirectPath, &m.FileLength,
		&m.MediaKey, &m.FileSHA256, &m.FileEncSHA256, &m.Direction, &status, &m.SentAt, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrMessageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("messageRepo.GetByID: %w", err)
	}
	m.InstanceID = instanceID.String
	m.Status = domain.MessageStatus(status)
	return m, nil
}

func (r *messageRepo) GetByWhatsAppID(ctx context.Context, tenantID, instanceID, whatsappID string) (*domain.Message, error) {
	q := `
		SELECT id, tenant_id, instance_id, whatsapp_id, phone, body, type, mime_type, file_name, media_url, direct_path, file_length, media_key, file_sha256, file_enc_sha256, direction, status, sent_at, created_at
		FROM messages WHERE tenant_id = $1 AND COALESCE(instance_id, '') = COALESCE($2, '') AND whatsapp_id = $3
		ORDER BY created_at DESC
		LIMIT 1
	`
	m := &domain.Message{}
	var status string
	var dbInstanceID sql.NullString
	err := r.db.QueryRowContext(ctx, q, tenantID, nullableString(instanceID), whatsappID).Scan(
		&m.ID, &m.TenantID, &dbInstanceID, &m.WhatsAppID, &m.Phone,
		&m.Body, &m.Type, &m.MimeType, &m.FileName, &m.MediaURL, &m.DirectPath, &m.FileLength,
		&m.MediaKey, &m.FileSHA256, &m.FileEncSHA256, &m.Direction, &status, &m.SentAt, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrMessageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("messageRepo.GetByWhatsAppID: %w", err)
	}
	m.InstanceID = dbInstanceID.String
	m.Status = domain.MessageStatus(status)
	return m, nil
}

func (r *messageRepo) ListByTenant(ctx context.Context, tenantID, instanceID string, limit, offset int) ([]domain.Message, error) {
	q := `
		SELECT id, tenant_id, instance_id, whatsapp_id, phone, body, type, mime_type, file_name, media_url, direct_path, file_length, media_key, file_sha256, file_enc_sha256, direction, status, sent_at, created_at
		FROM messages WHERE tenant_id = $1 AND ($2 = '' OR COALESCE(instance_id, '') = $2)
		ORDER BY created_at DESC LIMIT $3 OFFSET $4
	`
	rows, err := r.db.QueryContext(ctx, q, tenantID, instanceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("messageRepo.ListByTenant: %w", err)
	}
	defer rows.Close()

	var msgs []domain.Message
	for rows.Next() {
		m := domain.Message{}
		var status string
		var dbInstanceID sql.NullString
		if err := rows.Scan(
			&m.ID, &m.TenantID, &dbInstanceID, &m.WhatsAppID, &m.Phone,
			&m.Body, &m.Type, &m.MimeType, &m.FileName, &m.MediaURL, &m.DirectPath, &m.FileLength,
			&m.MediaKey, &m.FileSHA256, &m.FileEncSHA256, &m.Direction, &status, &m.SentAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.InstanceID = dbInstanceID.String
		m.Status = domain.MessageStatus(status)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func (r *messageRepo) ListConversations(ctx context.Context, tenantID, instanceID string, limit, offset int) ([]domain.Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.tenant_id, c.instance_id, c.phone, c.last_message_id, c.last_message_body, c.last_direction,
		       c.last_at, c.state, c.assigned_user_id, COALESCE(u.name, ''), c.note, c.unread_count, c.created_at, c.updated_at
		FROM conversations c
		LEFT JOIN users u ON u.id = NULLIF(c.assigned_user_id, '')
		WHERE c.tenant_id = $1 AND ($2 = '' OR c.instance_id = $2)
		ORDER BY c.last_at DESC
		LIMIT $3 OFFSET $4
	`, tenantID, instanceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("messageRepo.ListConversations: %w", err)
	}
	defer rows.Close()

	var items []domain.Conversation
	for rows.Next() {
		var item domain.Conversation
		var state string
		if err := rows.Scan(&item.ID, &item.TenantID, &item.InstanceID, &item.Phone, &item.LastMessageID, &item.LastMessageBody, &item.LastDirection, &item.LastAt, &state, &item.AssignedUserID, &item.AssignedName, &item.Note, &item.UnreadCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.State = domain.ConversationState(state)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *messageRepo) UpdateConversation(ctx context.Context, convo *domain.Conversation) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE conversations
		SET state = $1, assigned_user_id = $2, note = $3, updated_at = $4
		WHERE id = $5
	`, string(convo.State), convo.AssignedUserID, convo.Note, convo.UpdatedAt, convo.ID)
	if err != nil {
		return fmt.Errorf("messageRepo.UpdateConversation: %w", err)
	}
	return nil
}

func (r *messageRepo) GetConversation(ctx context.Context, tenantID, instanceID, phone string) (*domain.Conversation, error) {
	var item domain.Conversation
	var state string
	err := r.db.QueryRowContext(ctx, `
		SELECT c.id, c.tenant_id, c.instance_id, c.phone, c.last_message_id, c.last_message_body, c.last_direction,
		       c.last_at, c.state, c.assigned_user_id, COALESCE(u.name, ''), c.note, c.unread_count, c.created_at, c.updated_at
		FROM conversations c
		LEFT JOIN users u ON u.id = NULLIF(c.assigned_user_id, '')
		WHERE c.tenant_id = $1 AND c.instance_id = $2 AND c.phone = $3
	`, tenantID, instanceID, phone).Scan(&item.ID, &item.TenantID, &item.InstanceID, &item.Phone, &item.LastMessageID, &item.LastMessageBody, &item.LastDirection, &item.LastAt, &state, &item.AssignedUserID, &item.AssignedName, &item.Note, &item.UnreadCount, &item.CreatedAt, &item.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("messageRepo.GetConversation: %w", err)
	}
	item.State = domain.ConversationState(state)
	return &item, nil
}

func (r *messageRepo) UpdateStatus(ctx context.Context, id string, status domain.MessageStatus) error {
	q := `UPDATE messages SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, string(status), id)
	if err != nil {
		return fmt.Errorf("messageRepo.UpdateStatus: %w", err)
	}
	return nil
}

func (r *messageRepo) upsertConversation(ctx context.Context, m *domain.Message) error {
	if m.InstanceID == "" || m.Phone == "" {
		return nil
	}
	now := time.Now().UTC()
	unread := 0
	if m.Direction == "inbound" {
		unread = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO conversations (id, tenant_id, instance_id, phone, last_message_id, last_message_body, last_direction, last_at, state, unread_count, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'open',$9,$10,$11)
		ON CONFLICT (tenant_id, instance_id, phone) DO UPDATE
		SET last_message_id = EXCLUDED.last_message_id,
		    last_message_body = EXCLUDED.last_message_body,
		    last_direction = EXCLUDED.last_direction,
		    last_at = EXCLUDED.last_at,
		    unread_count = CASE WHEN EXCLUDED.last_direction = 'inbound' THEN conversations.unread_count + 1 ELSE conversations.unread_count END,
		    updated_at = EXCLUDED.updated_at
	`, "convo_"+m.InstanceID+"_"+m.Phone, m.TenantID, m.InstanceID, m.Phone, m.ID, m.Body, m.Direction, m.SentAt, unread, now, now)
	return err
}
