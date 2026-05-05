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
		INSERT INTO whatsapp_sessions (tenant_id, device_jid, phone, status, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id) DO UPDATE
		SET device_jid = EXCLUDED.device_jid,
		    phone = EXCLUDED.phone,
		    status = EXCLUDED.status,
		    updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, q,
		session.TenantID,
		session.DeviceJID,
		session.Phone,
		string(session.Status),
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("sessionRepo.Upsert: %w", err)
	}
	return nil
}

func (r *sessionRepo) GetByTenant(ctx context.Context, tenantID string) (*domain.Session, error) {
	q := `
		SELECT tenant_id, device_jid, phone, status, updated_at
		FROM whatsapp_sessions
		WHERE tenant_id = $1
	`
	session := &domain.Session{}
	var status string
	err := r.db.QueryRowContext(ctx, q, tenantID).Scan(
		&session.TenantID,
		&session.DeviceJID,
		&session.Phone,
		&status,
		&session.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrSessionMetadataNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.GetByTenant: %w", err)
	}
	session.Status = domain.SessionStatus(status)
	return session, nil
}

func (r *sessionRepo) ListTenantIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tenant_id FROM whatsapp_sessions`)
	if err != nil {
		return nil, fmt.Errorf("sessionRepo.ListTenantIDs: %w", err)
	}
	defer rows.Close()

	var tenantIDs []string
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			return nil, err
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	return tenantIDs, rows.Err()
}

func (r *sessionRepo) Delete(ctx context.Context, tenantID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM whatsapp_sessions WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return fmt.Errorf("sessionRepo.Delete: %w", err)
	}
	return nil
}

var _ domain.SessionRepository = (*sessionRepo)(nil)
