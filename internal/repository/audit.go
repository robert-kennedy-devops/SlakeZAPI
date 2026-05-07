package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/whatsapp-saas/api/internal/domain"
)

type auditLogRepo struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) domain.AuditLogRepository {
	return &auditLogRepo{db: db}
}

func (r *auditLogRepo) Create(ctx context.Context, item *domain.AuditLog) error {
	payload := []byte("{}")
	if len(item.Payload) > 0 {
		encoded, err := json.Marshal(item.Payload)
		if err != nil {
			return fmt.Errorf("auditLogRepo.Create marshal payload: %w", err)
		}
		payload = encoded
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, tenant_id, instance_id, user_id, request_id, action, resource, payload_json, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`,
		item.ID, item.TenantID, nullableString(item.InstanceID), nullableString(item.UserID), nullableString(item.RequestID),
		item.Action, item.Resource, payload, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("auditLogRepo.Create: %w", err)
	}
	return nil
}

func (r *auditLogRepo) List(ctx context.Context, tenantID, instanceID, actionPrefix string, limit int) ([]domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, COALESCE(instance_id, ''), COALESCE(user_id, ''), COALESCE(request_id, ''), action, resource, payload_json, created_at
		FROM audit_logs
		WHERE tenant_id = $1
		  AND ($2 = '' OR COALESCE(instance_id, '') = $2)
		  AND ($3 = '' OR action LIKE $3 || '%')
		ORDER BY created_at DESC
		LIMIT $4
	`, tenantID, instanceID, actionPrefix, limit)
	if err != nil {
		return nil, fmt.Errorf("auditLogRepo.List: %w", err)
	}
	defer rows.Close()

	items := make([]domain.AuditLog, 0, limit)
	for rows.Next() {
		var item domain.AuditLog
		var payloadBytes []byte
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.InstanceID,
			&item.UserID,
			&item.RequestID,
			&item.Action,
			&item.Resource,
			&payloadBytes,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("auditLogRepo.List scan: %w", err)
		}
		if len(payloadBytes) > 0 {
			_ = json.Unmarshal(payloadBytes, &item.Payload)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
