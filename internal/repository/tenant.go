package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/whatsapp-saas/api/internal/domain"
)

type tenantRepo struct {
	db *sql.DB
}

// NewTenantRepository returns a PostgreSQL-backed TenantRepository.
func NewTenantRepository(db *sql.DB) domain.TenantRepository {
	return &tenantRepo{db: db}
}

func (r *tenantRepo) Create(ctx context.Context, t *domain.Tenant) error {
	q := `
		INSERT INTO tenants (id, name, email, active, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q, t.ID, t.Name, t.Email, t.Active, t.CreatedAt)
	if err != nil {
		return fmt.Errorf("tenantRepo.Create: %w", err)
	}
	return nil
}

func (r *tenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	q := `SELECT id, name, email, active, created_at FROM tenants WHERE id = $1`
	row := r.db.QueryRowContext(ctx, q, id)

	t := &domain.Tenant{}
	err := row.Scan(&t.ID, &t.Name, &t.Email, &t.Active, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("tenantRepo.GetByID: %w", err)
	}
	return t, nil
}

func (r *tenantRepo) GetByEmail(ctx context.Context, email string) (*domain.Tenant, error) {
	q := `SELECT id, name, email, active, created_at FROM tenants WHERE email = $1`
	row := r.db.QueryRowContext(ctx, q, email)

	t := &domain.Tenant{}
	err := row.Scan(&t.ID, &t.Name, &t.Email, &t.Active, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTenantNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("tenantRepo.GetByEmail: %w", err)
	}
	return t, nil
}

func (r *tenantRepo) Update(ctx context.Context, t *domain.Tenant) error {
	q := `UPDATE tenants SET name=$1, email=$2, active=$3 WHERE id=$4`
	_, err := r.db.ExecContext(ctx, q, t.Name, t.Email, t.Active, t.ID)
	if err != nil {
		return fmt.Errorf("tenantRepo.Update: %w", err)
	}
	return nil
}
