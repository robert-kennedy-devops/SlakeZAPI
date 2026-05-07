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
		INSERT INTO user_sessions (id, user_id, token_hash, refresh_token_hash, expires_at, refresh_expires_at, created_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, q,
		session.ID, session.UserID, session.TokenHash, session.RefreshTokenHash, session.ExpiresAt, session.RefreshExpiresAt, session.CreatedAt, session.LastUsedAt,
	)
	if err != nil {
		return fmt.Errorf("userSessionRepo.Create: %w", err)
	}
	return nil
}

func (r *userSessionRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.UserSession, error) {
	q := `
		SELECT id, user_id, token_hash, refresh_token_hash, expires_at, refresh_expires_at, created_at, last_used_at
		FROM user_sessions
		WHERE token_hash = $1
	`
	row := r.db.QueryRowContext(ctx, q, tokenHash)

	session := &domain.UserSession{}
	err := row.Scan(
		&session.ID, &session.UserID, &session.TokenHash, &session.RefreshTokenHash, &session.ExpiresAt, &session.RefreshExpiresAt, &session.CreatedAt, &session.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrUserSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userSessionRepo.GetByHash: %w", err)
	}
	return session, nil
}

func (r *userSessionRepo) GetByRefreshHash(ctx context.Context, refreshHash string) (*domain.UserSession, error) {
	q := `
		SELECT id, user_id, token_hash, refresh_token_hash, expires_at, refresh_expires_at, created_at, last_used_at
		FROM user_sessions
		WHERE refresh_token_hash = $1
	`
	row := r.db.QueryRowContext(ctx, q, refreshHash)

	session := &domain.UserSession{}
	err := row.Scan(
		&session.ID, &session.UserID, &session.TokenHash, &session.RefreshTokenHash, &session.ExpiresAt, &session.RefreshExpiresAt, &session.CreatedAt, &session.LastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrUserSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userSessionRepo.GetByRefreshHash: %w", err)
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

func (r *userSessionRepo) Touch(ctx context.Context, id string, accessExpiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE user_sessions SET last_used_at = $1, expires_at = $2 WHERE id = $3`, time.Now().UTC(), accessExpiresAt, id)
	if err != nil {
		return fmt.Errorf("userSessionRepo.Touch: %w", err)
	}
	return nil
}

func (r *userSessionRepo) RotateTokens(ctx context.Context, id, accessHash, refreshHash string, accessExpiresAt, refreshExpiresAt time.Time) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE user_sessions SET token_hash = $1, refresh_token_hash = $2, expires_at = $3, refresh_expires_at = $4, last_used_at = $5 WHERE id = $6`,
		accessHash,
		refreshHash,
		accessExpiresAt,
		refreshExpiresAt,
		time.Now().UTC(),
		id,
	)
	if err != nil {
		return fmt.Errorf("userSessionRepo.RotateTokens: %w", err)
	}
	return nil
}
