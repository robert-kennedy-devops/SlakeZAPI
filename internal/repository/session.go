package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/whatsapp-saas/api/internal/domain"
)

type sessionRepo struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) domain.SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Upsert(ctx context.Context, session *domain.Session) error {
	q := `
		INSERT INTO whatsapp_sessions (instance_id, tenant_id, device_jid, phone, status, last_event, last_error, qr_code, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (instance_id) DO UPDATE
		SET tenant_id = EXCLUDED.tenant_id,
		    device_jid = EXCLUDED.device_jid,
		    phone = EXCLUDED.phone,
		    status = EXCLUDED.status,
		    last_event = EXCLUDED.last_event,
		    last_error = EXCLUDED.last_error,
		    qr_code = EXCLUDED.qr_code,
		    updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, q,
		session.InstanceID,
		session.TenantID,
		session.DeviceJID,
		session.Phone,
		string(session.Status),
		session.LastEvent,
		session.LastError,
		session.QRCode,
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("sessionRepo.Upsert: %w", err)
	}
	return nil
}

func (r *sessionRepo) GetByInstance(ctx context.Context, instanceID string) (*domain.Session, error) {
	q := `
		SELECT tenant_id, instance_id, device_jid, phone, status, last_event, last_error, qr_code, updated_at
		FROM whatsapp_sessions
		WHERE instance_id = $1
	`
	session := &domain.Session{}
	var status string
	err := r.db.QueryRowContext(ctx, q, instanceID).Scan(
		&session.TenantID,
		&session.InstanceID,
		&session.DeviceJID,
		&session.Phone,
		&status,
		&session.LastEvent,
		&session.LastError,
		&session.QRCode,
		&session.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrSessionMetadataNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.GetByInstance: %w", err)
	}
	session.Status = domain.SessionStatus(status)
	return session, nil
}

func (r *sessionRepo) ListSessions(ctx context.Context) ([]domain.Session, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id, instance_id, device_jid, phone, status, last_event, last_error, qr_code, updated_at FROM whatsapp_sessions`)
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.ListSessions: %w", err)
	}
	defer rows.Close()

	var sessions []domain.Session
	for rows.Next() {
		var session domain.Session
		var status string
		if err := rows.Scan(
			&session.TenantID,
			&session.InstanceID,
			&session.DeviceJID,
			&session.Phone,
			&status,
			&session.LastEvent,
			&session.LastError,
			&session.QRCode,
			&session.UpdatedAt,
		); err != nil {
			return nil, err
		}
		session.Status = domain.SessionStatus(status)
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

func (r *sessionRepo) Delete(ctx context.Context, instanceID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM whatsapp_sessions WHERE instance_id = $1`, instanceID)
	if err != nil {
		return fmt.Errorf("sessionRepo.Delete: %w", err)
	}
	return nil
}

var _ domain.SessionRepository = (*sessionRepo)(nil)
