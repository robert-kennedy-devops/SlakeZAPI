package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
)

type userRepo struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	q := `
		INSERT INTO users (id, email, name, password_hash, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, q,
		user.ID, user.Email, user.Name, user.PasswordHash, user.Active, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	q := `SELECT id, email, name, password_hash, active, created_at, updated_at FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, q, id)

	user := &domain.User{}
	err := row.Scan(
		&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Active, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByID: %w", err)
	}
	return user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := `SELECT id, email, name, password_hash, active, created_at, updated_at FROM users WHERE email = $1`
	row := r.db.QueryRowContext(ctx, q, email)

	user := &domain.User{}
	err := row.Scan(
		&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.Active, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByEmail: %w", err)
	}
	return user, nil
}

func (r *userRepo) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now().UTC()
	q := `UPDATE users SET email = $1, name = $2, password_hash = $3, active = $4, updated_at = $5 WHERE id = $6`
	_, err := r.db.ExecContext(ctx, q,
		user.Email, user.Name, user.PasswordHash, user.Active, user.UpdatedAt, user.ID,
	)
	if err != nil {
		return fmt.Errorf("userRepo.Update: %w", err)
	}
	return nil
}
