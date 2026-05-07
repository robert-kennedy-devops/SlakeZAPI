package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
)

type instanceRepo struct {
	db *sql.DB
}

func NewInstanceRepository(db *sql.DB) domain.InstanceRepository {
	return &instanceRepo{db: db}
}

func (r *instanceRepo) Create(ctx context.Context, instance *domain.Instance) error {
	q := `
		INSERT INTO instances (id, tenant_id, name, phone, status, is_default, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	_, err := r.db.ExecContext(ctx, q,
		instance.ID, instance.TenantID, instance.Name, instance.Phone, string(instance.Status), instance.IsDefault, instance.CreatedAt, instance.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("instanceRepo.Create: %w", err)
	}
	return nil
}

func (r *instanceRepo) GetByID(ctx context.Context, id string) (*domain.Instance, error) {
	q := `SELECT id, tenant_id, name, phone, status, is_default, created_at, updated_at FROM instances WHERE id = $1`
	var instance domain.Instance
	var status string
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&instance.ID, &instance.TenantID, &instance.Name, &instance.Phone, &status, &instance.IsDefault, &instance.CreatedAt, &instance.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrInstanceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("instanceRepo.GetByID: %w", err)
	}
	instance.Status = domain.SessionStatus(status)
	return &instance, nil
}

func (r *instanceRepo) GetDefaultByTenant(ctx context.Context, tenantID string) (*domain.Instance, error) {
	q := `SELECT id, tenant_id, name, phone, status, is_default, created_at, updated_at FROM instances WHERE tenant_id = $1 AND is_default = true LIMIT 1`
	var instance domain.Instance
	var status string
	err := r.db.QueryRowContext(ctx, q, tenantID).Scan(
		&instance.ID, &instance.TenantID, &instance.Name, &instance.Phone, &status, &instance.IsDefault, &instance.CreatedAt, &instance.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrInstanceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("instanceRepo.GetDefaultByTenant: %w", err)
	}
	instance.Status = domain.SessionStatus(status)
	return &instance, nil
}

func (r *instanceRepo) ListByTenant(ctx context.Context, tenantID string) ([]domain.Instance, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, tenant_id, name, phone, status, is_default, created_at, updated_at FROM instances WHERE tenant_id = $1 ORDER BY is_default DESC, created_at ASC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("instanceRepo.ListByTenant: %w", err)
	}
	defer rows.Close()

	var instances []domain.Instance
	for rows.Next() {
		var instance domain.Instance
		var status string
		if err := rows.Scan(&instance.ID, &instance.TenantID, &instance.Name, &instance.Phone, &status, &instance.IsDefault, &instance.CreatedAt, &instance.UpdatedAt); err != nil {
			return nil, err
		}
		instance.Status = domain.SessionStatus(status)
		instances = append(instances, instance)
	}
	return instances, rows.Err()
}

func (r *instanceRepo) UpdateStatus(ctx context.Context, instanceID string, status domain.SessionStatus, phone string, updatedAt time.Time) error {
	q := `UPDATE instances SET status = $1, phone = $2, updated_at = $3 WHERE id = $4`
	if _, err := r.db.ExecContext(ctx, q, string(status), phone, updatedAt, instanceID); err != nil {
		return fmt.Errorf("instanceRepo.UpdateStatus: %w", err)
	}
	return nil
}

var _ domain.InstanceRepository = (*instanceRepo)(nil)
