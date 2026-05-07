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

func (r *tenantUserRepo) GetByID(ctx context.Context, id string) (*domain.TenantUser, error) {
	q := `SELECT id, tenant_id, user_id, role, created_at FROM tenant_users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, q, id)

	tenantUser := &domain.TenantUser{}
	var role string
	err := row.Scan(&tenantUser.ID, &tenantUser.TenantID, &tenantUser.UserID, &role, &tenantUser.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrTenantAccessDenied
	}
	if err != nil {
		return nil, fmt.Errorf("tenantUserRepo.GetByID: %w", err)
	}
	tenantUser.Role = domain.UserRole(role)
	return tenantUser, nil
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

func (r *tenantUserRepo) ListByTenant(ctx context.Context, tenantID string) ([]domain.TenantMember, error) {
	q := `
		SELECT tu.id, tu.tenant_id, tu.user_id, u.email, u.name, tu.role, u.active, tu.created_at
		FROM tenant_users tu
		INNER JOIN users u ON u.id = tu.user_id
		WHERE tu.tenant_id = $1
		ORDER BY tu.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("tenantUserRepo.ListByTenant: %w", err)
	}
	defer rows.Close()

	var members []domain.TenantMember
	for rows.Next() {
		var member domain.TenantMember
		var role string
		if err := rows.Scan(
			&member.ID, &member.TenantID, &member.UserID, &member.Email, &member.Name, &role, &member.Active, &member.CreatedAt,
		); err != nil {
			return nil, err
		}
		member.Role = domain.UserRole(role)
		members = append(members, member)
	}
	return members, rows.Err()
}

func (r *tenantUserRepo) UpdateRole(ctx context.Context, id string, role domain.UserRole) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tenant_users SET role = $1 WHERE id = $2`, string(role), id)
	if err != nil {
		return fmt.Errorf("tenantUserRepo.UpdateRole: %w", err)
	}
	return nil
}
