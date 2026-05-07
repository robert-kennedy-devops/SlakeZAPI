package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

const defaultUserSessionTTL = 24 * time.Hour

type UserAuthUsecase struct {
	userRepo        domain.UserRepository
	tenantRepo      domain.TenantRepository
	tenantUserRepo  domain.TenantUserRepository
	userSessionRepo domain.UserSessionRepository
	subRepo         domain.SubscriptionRepository
	log             *logger.Logger
}

func NewUserAuthUsecase(
	userRepo domain.UserRepository,
	tenantRepo domain.TenantRepository,
	tenantUserRepo domain.TenantUserRepository,
	userSessionRepo domain.UserSessionRepository,
	subRepo domain.SubscriptionRepository,
	log *logger.Logger,
) *UserAuthUsecase {
	return &UserAuthUsecase{
		userRepo:        userRepo,
		tenantRepo:      tenantRepo,
		tenantUserRepo:  tenantUserRepo,
		userSessionRepo: userSessionRepo,
		subRepo:         subRepo,
		log:             log,
	}
}

func (u *UserAuthUsecase) SignUp(ctx context.Context, req domain.SignUpRequest) (*domain.AuthSessionResponse, error) {
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("%w: name, email and password are required", domain.ErrBadRequest)
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("%w: password must be at least 8 characters", domain.ErrBadRequest)
	}
	if req.TenantName == "" {
		return nil, fmt.Errorf("%w: tenant_name is required", domain.ErrBadRequest)
	}
	if req.Plan == "" {
		req.Plan = domain.PlanStarter
	}
	plan, ok := domain.PlanByName(req.Plan)
	if !ok {
		return nil, fmt.Errorf("%w: invalid plan", domain.ErrBadRequest)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: string(passwordHash),
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := u.userRepo.Create(ctx, user); err != nil {
		if isUserUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, err
	}

	tenant := &domain.Tenant{
		ID:        uuid.NewString(),
		Name:      req.TenantName,
		Email:     req.Email,
		CreatedAt: now,
		Active:    true,
	}
	if err := u.tenantRepo.Create(ctx, tenant); err != nil {
		if isUserUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, err
	}

	membership := &domain.TenantUser{
		ID:        uuid.NewString(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		Role:      domain.UserRoleOwner,
		CreatedAt: now,
	}
	if err := u.tenantUserRepo.Create(ctx, membership); err != nil {
		if isUserUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, err
	}

	sub := &domain.Subscription{
		ID:        uuid.NewString(),
		TenantID:  tenant.ID,
		PlanID:    plan.ID,
		Status:    "active",
		PeriodEnd: now.Add(30 * 24 * time.Hour),
		CreatedAt: now,
	}
	if err := u.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}

	return u.createSession(ctx, user, tenant, membership)
}

func (u *UserAuthUsecase) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthSessionResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("%w: email and password are required", domain.ErrBadRequest)
	}

	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	if !user.Active {
		return nil, domain.ErrUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	memberships, err := u.tenantUserRepo.ListByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, domain.ErrTenantAccessDenied
	}
	membership := &memberships[0]
	tenant, err := u.tenantRepo.GetByID(ctx, membership.TenantID)
	if err != nil {
		return nil, err
	}

	return u.createSession(ctx, user, tenant, membership)
}

func (u *UserAuthUsecase) GetCurrentUser(ctx context.Context, userID, tenantID string) (*domain.CurrentUserResponse, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.Active {
		return nil, domain.ErrUserInactive
	}

	memberships, err := u.tenantUserRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, domain.ErrTenantAccessDenied
	}

	selected := memberships[0]
	if tenantID != "" {
		for _, membership := range memberships {
			if membership.TenantID == tenantID {
				selected = membership
				break
			}
		}
	}
	tenant, err := u.tenantRepo.GetByID(ctx, selected.TenantID)
	if err != nil {
		return nil, err
	}

	return &domain.CurrentUserResponse{
		User:        user,
		Tenant:      tenant,
		Membership:  &selected,
		Memberships: memberships,
	}, nil
}

func (u *UserAuthUsecase) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return domain.ErrUnauthorized
	}
	return u.userSessionRepo.DeleteByID(ctx, sessionID)
}

func (u *UserAuthUsecase) ValidateSessionToken(ctx context.Context, token string) (*domain.UserSession, error) {
	if token == "" {
		return nil, domain.ErrUnauthorized
	}
	hash := hashSessionToken(token)
	session, err := u.userSessionRepo.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrUserSessionNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = u.userSessionRepo.DeleteByID(ctx, session.ID)
		return nil, domain.ErrUserSessionExpired
	}
	go func(id string) {
		_ = u.userSessionRepo.UpdateLastUsed(context.Background(), id)
	}(session.ID)
	return session, nil
}

func (u *UserAuthUsecase) createSession(ctx context.Context, user *domain.User, tenant *domain.Tenant, membership *domain.TenantUser) (*domain.AuthSessionResponse, error) {
	rawToken, err := generateRawToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}
	now := time.Now().UTC()
	session := &domain.UserSession{
		ID:         uuid.NewString(),
		UserID:     user.ID,
		TokenHash:  hashSessionToken(rawToken),
		ExpiresAt:  now.Add(defaultUserSessionTTL),
		CreatedAt:  now,
		LastUsedAt: now,
	}
	if err := u.userSessionRepo.Create(ctx, session); err != nil {
		if isUserUniqueViolation(err) {
			return nil, domain.ErrConflict
		}
		return nil, err
	}

	u.log.WithContext(ctx).Info("user session created", map[string]interface{}{
		"user_id":    user.ID,
		"tenant_id":  tenant.ID,
		"session_id": session.ID,
	})

	return &domain.AuthSessionResponse{
		Token:      rawToken,
		ExpiresAt:  session.ExpiresAt,
		User:       user,
		Tenant:     tenant,
		Membership: membership,
	}, nil
}

func generateRawToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func isUserUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
