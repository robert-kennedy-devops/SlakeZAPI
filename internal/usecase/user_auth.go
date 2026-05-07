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

type UserAuthUsecase struct {
	userRepo        domain.UserRepository
	tenantRepo      domain.TenantRepository
	tenantUserRepo  domain.TenantUserRepository
	userSessionRepo domain.UserSessionRepository
	subRepo         domain.SubscriptionRepository
	accessTTL       time.Duration
	refreshTTL      time.Duration
	log             *logger.Logger
}

func NewUserAuthUsecase(
	userRepo domain.UserRepository,
	tenantRepo domain.TenantRepository,
	tenantUserRepo domain.TenantUserRepository,
	userSessionRepo domain.UserSessionRepository,
	subRepo domain.SubscriptionRepository,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	log *logger.Logger,
) *UserAuthUsecase {
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	return &UserAuthUsecase{
		userRepo:        userRepo,
		tenantRepo:      tenantRepo,
		tenantUserRepo:  tenantUserRepo,
		userSessionRepo: userSessionRepo,
		subRepo:         subRepo,
		accessTTL:       accessTTL,
		refreshTTL:      refreshTTL,
		log:             log,
	}
}

func (u *UserAuthUsecase) SignUp(ctx context.Context, req domain.SignUpRequest) (*domain.AuthSessionResponse, string, error) {
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, "", fmt.Errorf("%w: name, email and password are required", domain.ErrBadRequest)
	}
	if len(req.Password) < 8 {
		return nil, "", fmt.Errorf("%w: password must be at least 8 characters", domain.ErrBadRequest)
	}
	if req.TenantName == "" {
		return nil, "", fmt.Errorf("%w: tenant_name is required", domain.ErrBadRequest)
	}
	if req.Plan == "" {
		req.Plan = domain.PlanStarter
	}
	plan, ok := domain.PlanByName(req.Plan)
	if !ok {
		return nil, "", fmt.Errorf("%w: invalid plan", domain.ErrBadRequest)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
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
			return nil, "", domain.ErrConflict
		}
		return nil, "", err
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
			return nil, "", domain.ErrConflict
		}
		return nil, "", err
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
			return nil, "", domain.ErrConflict
		}
		return nil, "", err
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
		return nil, "", err
	}

	return u.createSession(ctx, user, tenant, membership)
}

func (u *UserAuthUsecase) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthSessionResponse, string, error) {
	if req.Email == "" || req.Password == "" {
		return nil, "", fmt.Errorf("%w: email and password are required", domain.ErrBadRequest)
	}

	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, "", domain.ErrInvalidCredentials
		}
		return nil, "", err
	}
	if !user.Active {
		return nil, "", domain.ErrUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", domain.ErrInvalidCredentials
	}

	memberships, err := u.tenantUserRepo.ListByUser(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	if len(memberships) == 0 {
		return nil, "", domain.ErrTenantAccessDenied
	}
	membership := &memberships[0]
	tenant, err := u.tenantRepo.GetByID(ctx, membership.TenantID)
	if err != nil {
		return nil, "", err
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

func (u *UserAuthUsecase) RefreshSession(ctx context.Context, refreshToken, tenantID string) (*domain.AuthSessionResponse, string, error) {
	if refreshToken == "" {
		return nil, "", domain.ErrUnauthorized
	}

	session, err := u.userSessionRepo.GetByRefreshHash(ctx, hashSessionToken(refreshToken))
	if err != nil {
		if errors.Is(err, domain.ErrUserSessionNotFound) {
			return nil, "", domain.ErrUnauthorized
		}
		return nil, "", err
	}
	now := time.Now().UTC()
	if now.After(session.RefreshExpiresAt) {
		_ = u.userSessionRepo.DeleteByID(ctx, session.ID)
		return nil, "", domain.ErrUserSessionExpired
	}

	user, err := u.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, "", err
	}
	if !user.Active {
		return nil, "", domain.ErrUserInactive
	}

	current, err := u.GetCurrentUser(ctx, session.UserID, tenantID)
	if err != nil {
		return nil, "", err
	}
	accessToken, err := generateRawToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}
	nextRefreshToken, err := generateRawToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}
	accessExpiresAt := now.Add(u.accessTTL)
	refreshExpiresAt := now.Add(u.refreshTTL)
	if err := u.userSessionRepo.RotateTokens(
		ctx,
		session.ID,
		hashSessionToken(accessToken),
		hashSessionToken(nextRefreshToken),
		accessExpiresAt,
		refreshExpiresAt,
	); err != nil {
		if isUserUniqueViolation(err) {
			return nil, "", domain.ErrConflict
		}
		return nil, "", err
	}

	return &domain.AuthSessionResponse{
		Token:            accessToken,
		ExpiresAt:        accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		User:             user,
		Tenant:           current.Tenant,
		Membership:       current.Membership,
	}, nextRefreshToken, nil
}

func (u *UserAuthUsecase) ListTenantMembers(ctx context.Context, actorUserID, tenantID string) ([]domain.TenantMember, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", domain.ErrBadRequest)
	}
	if _, err := u.tenantUserRepo.GetByUserAndTenant(ctx, actorUserID, tenantID); err != nil {
		return nil, err
	}
	return u.tenantUserRepo.ListByTenant(ctx, tenantID)
}

func (u *UserAuthUsecase) AddTenantMember(ctx context.Context, actorUserID, tenantID string, req domain.AddTenantMemberRequest) (*domain.TenantMember, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", domain.ErrBadRequest)
	}
	if req.Email == "" {
		return nil, fmt.Errorf("%w: email is required", domain.ErrBadRequest)
	}
	if !isManageableRole(req.Role) {
		return nil, fmt.Errorf("%w: invalid role", domain.ErrBadRequest)
	}

	actorMembership, err := u.tenantUserRepo.GetByUserAndTenant(ctx, actorUserID, tenantID)
	if err != nil {
		return nil, err
	}
	if err := validateRoleManagement(actorMembership.Role, req.Role, ""); err != nil {
		return nil, err
	}

	user, err := u.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if !user.Active {
		return nil, domain.ErrUserInactive
	}
	if _, err := u.tenantUserRepo.GetByUserAndTenant(ctx, user.ID, tenantID); err == nil {
		return nil, domain.ErrUserAlreadyInTenant
	} else if !errors.Is(err, domain.ErrTenantAccessDenied) {
		return nil, err
	}

	membership := &domain.TenantUser{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		UserID:    user.ID,
		Role:      req.Role,
		CreatedAt: time.Now().UTC(),
	}
	if err := u.tenantUserRepo.Create(ctx, membership); err != nil {
		if isUserUniqueViolation(err) {
			return nil, domain.ErrUserAlreadyInTenant
		}
		return nil, err
	}

	u.log.WithContext(ctx).Audit("app.members.add", map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   user.ID,
		"role":      req.Role,
	})

	return u.findTenantMember(ctx, tenantID, membership.ID)
}

func (u *UserAuthUsecase) UpdateTenantMemberRole(ctx context.Context, actorUserID, tenantID, memberID string, req domain.UpdateTenantMemberRoleRequest) (*domain.TenantMember, error) {
	if tenantID == "" || memberID == "" {
		return nil, fmt.Errorf("%w: tenant_id and member id are required", domain.ErrBadRequest)
	}
	if !isManageableRole(req.Role) {
		return nil, fmt.Errorf("%w: invalid role", domain.ErrBadRequest)
	}

	actorMembership, err := u.tenantUserRepo.GetByUserAndTenant(ctx, actorUserID, tenantID)
	if err != nil {
		return nil, err
	}
	targetMembership, err := u.tenantUserRepo.GetByID(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if targetMembership.TenantID != tenantID {
		return nil, domain.ErrTenantAccessDenied
	}
	if targetMembership.UserID == actorUserID {
		return nil, fmt.Errorf("%w: cannot change your own role", domain.ErrBadRequest)
	}
	if err := validateRoleManagement(actorMembership.Role, req.Role, targetMembership.Role); err != nil {
		return nil, err
	}

	if err := u.tenantUserRepo.UpdateRole(ctx, memberID, req.Role); err != nil {
		return nil, err
	}

	u.log.WithContext(ctx).Audit("app.members.role_update", map[string]interface{}{
		"tenant_id":  tenantID,
		"member_id":  memberID,
		"target_uid": targetMembership.UserID,
		"new_role":   req.Role,
	})

	return u.findTenantMember(ctx, tenantID, memberID)
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
	go func(id string, nextExpiry time.Time) {
		_ = u.userSessionRepo.Touch(context.Background(), id, nextExpiry)
	}(session.ID, time.Now().UTC().Add(u.accessTTL))
	return session, nil
}

func (u *UserAuthUsecase) createSession(ctx context.Context, user *domain.User, tenant *domain.Tenant, membership *domain.TenantUser) (*domain.AuthSessionResponse, string, error) {
	rawToken, err := generateRawToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("generate session token: %w", err)
	}
	refreshToken, err := generateRawToken(32)
	if err != nil {
		return nil, "", fmt.Errorf("generate refresh token: %w", err)
	}
	now := time.Now().UTC()
	session := &domain.UserSession{
		ID:               uuid.NewString(),
		UserID:           user.ID,
		TokenHash:        hashSessionToken(rawToken),
		RefreshTokenHash: hashSessionToken(refreshToken),
		ExpiresAt:        now.Add(u.accessTTL),
		RefreshExpiresAt: now.Add(u.refreshTTL),
		CreatedAt:        now,
		LastUsedAt:       now,
	}
	if err := u.userSessionRepo.Create(ctx, session); err != nil {
		if isUserUniqueViolation(err) {
			return nil, "", domain.ErrConflict
		}
		return nil, "", err
	}

	u.log.WithContext(ctx).Info("user session created", map[string]interface{}{
		"user_id":    user.ID,
		"tenant_id":  tenant.ID,
		"session_id": session.ID,
	})

	return &domain.AuthSessionResponse{
		Token:            rawToken,
		ExpiresAt:        session.ExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		User:             user,
		Tenant:           tenant,
		Membership:       membership,
	}, refreshToken, nil
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

func (u *UserAuthUsecase) findTenantMember(ctx context.Context, tenantID, memberID string) (*domain.TenantMember, error) {
	members, err := u.tenantUserRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		if member.ID == memberID {
			copied := member
			return &copied, nil
		}
	}
	return nil, domain.ErrTenantAccessDenied
}

func isManageableRole(role domain.UserRole) bool {
	switch role {
	case domain.UserRoleOwner, domain.UserRoleAdmin, domain.UserRoleOperator, domain.UserRoleViewer:
		return true
	default:
		return false
	}
}

func validateRoleManagement(actorRole, desiredRole, targetRole domain.UserRole) error {
	switch actorRole {
	case domain.UserRoleOwner:
		if desiredRole == "" {
			return nil
		}
		return nil
	case domain.UserRoleAdmin:
		if targetRole == domain.UserRoleOwner || targetRole == domain.UserRoleAdmin {
			return domain.ErrUserRoleForbidden
		}
		if desiredRole == domain.UserRoleOwner || desiredRole == domain.UserRoleAdmin {
			return domain.ErrUserRoleForbidden
		}
		return nil
	default:
		return domain.ErrUserRoleForbidden
	}
}
