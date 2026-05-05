package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
)

type apiKeyRepo struct {
	db *sql.DB
}

// NewAPIKeyRepository returns a PostgreSQL-backed APIKeyRepository.
func NewAPIKeyRepository(db *sql.DB) domain.APIKeyRepository {
	return &apiKeyRepo{db: db}
}

func (r *apiKeyRepo) Create(ctx context.Context, k *domain.APIKey) error {
	q := `
		INSERT INTO api_keys (id, tenant_id, key_hash, key_prefix, label, active, created_at, last_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, q,
		k.ID, k.TenantID, k.KeyHash, k.KeyPrefix, k.Label,
		k.Active, k.CreatedAt, k.LastUsed,
	)
	if err != nil {
		return fmt.Errorf("apiKeyRepo.Create: %w", err)
	}
	return nil
}

func (r *apiKeyRepo) GetByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	q := `
		SELECT id, tenant_id, key_hash, key_prefix, label, active, created_at, last_used
		FROM api_keys WHERE key_hash = $1 AND active = true
	`
	row := r.db.QueryRowContext(ctx, q, hash)

	k := &domain.APIKey{}
	err := row.Scan(&k.ID, &k.TenantID, &k.KeyHash, &k.KeyPrefix,
		&k.Label, &k.Active, &k.CreatedAt, &k.LastUsed)
	if err == sql.ErrNoRows {
		return nil, domain.ErrInvalidAPIKey
	}
	if err != nil {
		return nil, fmt.Errorf("apiKeyRepo.GetByHash: %w", err)
	}
	return k, nil
}

func (r *apiKeyRepo) ListByTenant(ctx context.Context, tenantID string) ([]domain.APIKey, error) {
	q := `
		SELECT id, tenant_id, key_hash, key_prefix, label, active, created_at, last_used
		FROM api_keys WHERE tenant_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("apiKeyRepo.ListByTenant: %w", err)
	}
	defer rows.Close()

	var keys []domain.APIKey
	for rows.Next() {
		k := domain.APIKey{}
		if err := rows.Scan(&k.ID, &k.TenantID, &k.KeyHash, &k.KeyPrefix,
			&k.Label, &k.Active, &k.CreatedAt, &k.LastUsed); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *apiKeyRepo) GetByID(ctx context.Context, keyID string) (*domain.APIKey, error) {
	q := `
		SELECT id, tenant_id, key_hash, key_prefix, label, active, created_at, last_used
		FROM api_keys WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, q, keyID)
	k := &domain.APIKey{}
	err := row.Scan(&k.ID, &k.TenantID, &k.KeyHash, &k.KeyPrefix,
		&k.Label, &k.Active, &k.CreatedAt, &k.LastUsed)
	if err == sql.ErrNoRows {
		return nil, domain.ErrInvalidAPIKey
	}
	if err != nil {
		return nil, fmt.Errorf("apiKeyRepo.GetByID: %w", err)
	}
	return k, nil
}

func (r *apiKeyRepo) UpdateLastUsed(ctx context.Context, keyID string) error {
	q := `UPDATE api_keys SET last_used = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, time.Now().UTC(), keyID)
	return err
}

func (r *apiKeyRepo) Revoke(ctx context.Context, keyID string) error {
	q := `UPDATE api_keys SET active = false WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, keyID)
	return err
}
