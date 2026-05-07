package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/whatsapp-saas/api/internal/domain"
)

type tenantUserRepo struct {
	db *sql.DB
}

func NewTenantUserRepository(db *sql.DB) domain.TenantUserRepository {
	return &tenantUserRepo{db: db}
}

func (r *tenantUserRepo) Create(ctx context.Context, tenantUser *domain.TenantUser) error {
	q := `
		INSERT INTO tenant_users (id, tenant_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q,
		tenantUser.ID, tenantUser.TenantID, tenantUser.UserID, string(tenantUser.Role), tenantUser.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("tenantUserRepo.Create: %w", err)
	}
	return nil
}

func (r *tenantUserRepo) GetByUserAndTenant(ctx context.Context, userID, tenantID string) (*domain.TenantUser, error) {
	q := `SELECT id, tenant_id, user_id, role, created_at FROM tenant_users WHERE user_id = $1 AND tenant_id = $2`
	row := r.db.QueryRowContext(ctx, q, userID, tenantID)

	tenantUser := &domain.TenantUser{}
	var role string
	err := row.Scan(&tenantUser.ID, &tenantUser.TenantID, &tenantUser.UserID, &role, &tenantUser.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTenantAccessDenied
	}
	if err != nil {
		return nil, fmt.Errorf("tenantUserRepo.GetByUserAndTenant: %w", err)
	}
	tenantUser.Role = domain.UserRole(role)
	return tenantUser, nil
}

func (r *tenantUserRepo) ListByUser(ctx context.Context, userID string) ([]domain.TenantUser, error) {
	q := `SELECT id, tenant_id, user_id, role, created_at FROM tenant_users WHERE user_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("tenantUserRepo.ListByUser: %w", err)
	}
	defer rows.Close()

	var memberships []domain.TenantUser
	for rows.Next() {
		var membership domain.TenantUser
		var role string
		if err := rows.Scan(&membership.ID, &membership.TenantID, &membership.UserID, &role, &membership.CreatedAt); err != nil {
			return nil, err
		}
		membership.Role = domain.UserRole(role)
		memberships = append(memberships, membership)
	}
	return memberships, rows.Err()
}
