package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
)

type userSessionRepo struct {
	db *sql.DB
}

func NewUserSessionRepository(db *sql.DB) domain.UserSessionRepository {
	return &userSessionRepo{db: db}
}

func (r *userSessionRepo) Create(ctx context.Context, session *domain.UserSession) error {
	q := `
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, q,
		session.ID, session.UserID, session.TokenHash, session.ExpiresAt, session.CreatedAt, session.LastUsedAt,
	)
	if err != nil {
		return fmt.Errorf("userSessionRepo.Create: %w", err)
	}
	return nil
}

func (r *userSessionRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.UserSession, error) {
	q := `
		SELECT id, user_id, token_hash, expires_at, created_at, last_used_at
		FROM user_sessions
		WHERE token_hash = $1
	`
	row := r.db.QueryRowContext(ctx, q, tokenHash)

	session := &domain.UserSession{}
	err := row.Scan(
		&session.ID, &session.UserID, &session.TokenHash, &session.ExpiresAt, &session.CreatedAt, &session.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrUserSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userSessionRepo.GetByHash: %w", err)
	}
	return session, nil
}

func (r *userSessionRepo) DeleteByID(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("userSessionRepo.DeleteByID: %w", err)
	}
	return nil
}

func (r *userSessionRepo) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("userSessionRepo.DeleteByUser: %w", err)
	}
	return nil
}

func (r *userSessionRepo) UpdateLastUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE user_sessions SET last_used_at = $1 WHERE id = $2`, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("userSessionRepo.UpdateLastUsed: %w", err)
	}
	return nil
}
