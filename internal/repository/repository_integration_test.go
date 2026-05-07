package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/testutil"
)

func TestAPIKeyRepositoryLifecycle(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	tenantRepo := NewTenantRepository(db)
	apiKeyRepo := NewAPIKeyRepository(db)

	tenant := &domain.Tenant{
		ID:        uuid.NewString(),
		Name:      "Tenant Repo",
		Email:     "repo@example.com",
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}
	if err := tenantRepo.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	key := &domain.APIKey{
		ID:        uuid.NewString(),
		TenantID:  tenant.ID,
		KeyHash:   "hash-1",
		KeyPrefix: "prefix-1",
		Label:     "default",
		Active:    true,
		CreatedAt: time.Now().UTC(),
		LastUsed:  time.Now().UTC(),
	}
	if err := apiKeyRepo.Create(ctx, key); err != nil {
		t.Fatalf("create api key: %v", err)
	}

	got, err := apiKeyRepo.GetByHash(ctx, key.KeyHash)
	if err != nil {
		t.Fatalf("get api key by hash: %v", err)
	}
	if got.ID != key.ID || got.TenantID != tenant.ID {
		t.Fatalf("unexpected api key loaded: %+v", got)
	}

	if err := apiKeyRepo.Revoke(ctx, key.ID); err != nil {
		t.Fatalf("revoke api key: %v", err)
	}
	if _, err := apiKeyRepo.GetByHash(ctx, key.KeyHash); err != domain.ErrInvalidAPIKey {
		t.Fatalf("expected invalid api key after revoke, got %v", err)
	}
}

func TestUsageRepositoryTracksCurrentMonth(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	tenantRepo := NewTenantRepository(db)
	usageRepo := NewUsageRepository(db)

	tenant := &domain.Tenant{
		ID:        uuid.NewString(),
		Name:      "Tenant Usage",
		Email:     "usage@example.com",
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}
	if err := tenantRepo.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	if err := usageRepo.IncrementSent(ctx, tenant.ID); err != nil {
		t.Fatalf("increment sent: %v", err)
	}
	if err := usageRepo.IncrementReceived(ctx, tenant.ID); err != nil {
		t.Fatalf("increment received: %v", err)
	}

	usage, err := usageRepo.GetCurrentMonth(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("get usage: %v", err)
	}
	if usage.Sent != 1 || usage.Received != 1 {
		t.Fatalf("unexpected usage counters: %+v", usage)
	}
}

func TestMessageRepositoryPersistsMediaFields(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	tenantRepo := NewTenantRepository(db)
	msgRepo := NewMessageRepository(db)

	tenant := &domain.Tenant{
		ID:        uuid.NewString(),
		Name:      "Tenant Message",
		Email:     "message@example.com",
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}
	if err := tenantRepo.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	now := time.Now().UTC()
	msg := &domain.Message{
		ID:            uuid.NewString(),
		TenantID:      tenant.ID,
		WhatsAppID:    "wa-media-123",
		Phone:         "5511999999999",
		Body:          "documento",
		Type:          "document",
		MimeType:      "application/pdf",
		FileName:      "contrato.pdf",
		MediaURL:      "https://mmg.whatsapp.net/media",
		DirectPath:    "/v/t62.7118-24/example",
		FileLength:    128,
		MediaKey:      []byte("media-key"),
		FileSHA256:    []byte("file-sha"),
		FileEncSHA256: []byte("file-enc"),
		Direction:     "inbound",
		Status:        domain.MessageStatusDelivered,
		SentAt:        now,
		CreatedAt:     now,
	}
	if err := msgRepo.Create(ctx, msg); err != nil {
		t.Fatalf("create message: %v", err)
	}

	got, err := msgRepo.GetByID(ctx, msg.ID)
	if err != nil {
		t.Fatalf("get message: %v", err)
	}
	if got.DirectPath != msg.DirectPath || got.MediaURL != msg.MediaURL || got.FileLength != msg.FileLength {
		t.Fatalf("unexpected message metadata: %+v", got)
	}
	if string(got.MediaKey) != string(msg.MediaKey) || string(got.FileSHA256) != string(msg.FileSHA256) || string(got.FileEncSHA256) != string(msg.FileEncSHA256) {
		t.Fatalf("unexpected message media keys: %+v", got)
	}
}

func TestUserRepositoriesLifecycle(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	userRepo := NewUserRepository(db)
	tenantRepo := NewTenantRepository(db)
	tenantUserRepo := NewTenantUserRepository(db)
	userSessionRepo := NewUserSessionRepository(db)

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        "owner@example.com",
		Name:         "Owner User",
		PasswordHash: "hashed-password",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	gotUser, err := userRepo.GetByEmail(ctx, user.Email)
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if gotUser.ID != user.ID || gotUser.Name != user.Name {
		t.Fatalf("unexpected user loaded: %+v", gotUser)
	}

	tenant := &domain.Tenant{
		ID:        uuid.NewString(),
		Name:      "Tenant Auth",
		Email:     "tenant-auth@example.com",
		Active:    true,
		CreatedAt: now,
	}
	if err := tenantRepo.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	membership := &domain.TenantUser{
		ID:        uuid.NewString(),
		TenantID:  tenant.ID,
		UserID:    user.ID,
		Role:      domain.UserRoleOwner,
		CreatedAt: now,
	}
	if err := tenantUserRepo.Create(ctx, membership); err != nil {
		t.Fatalf("create tenant user: %v", err)
	}

	gotMembership, err := tenantUserRepo.GetByUserAndTenant(ctx, user.ID, tenant.ID)
	if err != nil {
		t.Fatalf("get tenant membership: %v", err)
	}
	if gotMembership.Role != domain.UserRoleOwner {
		t.Fatalf("unexpected membership loaded: %+v", gotMembership)
	}

	session := &domain.UserSession{
		ID:               uuid.NewString(),
		UserID:           user.ID,
		TokenHash:        "token-hash-1",
		RefreshTokenHash: "refresh-hash-1",
		ExpiresAt:        now.Add(24 * time.Hour),
		RefreshExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedAt:        now,
		LastUsedAt:       now,
	}
	if err := userSessionRepo.Create(ctx, session); err != nil {
		t.Fatalf("create user session: %v", err)
	}

	gotSession, err := userSessionRepo.GetByHash(ctx, session.TokenHash)
	if err != nil {
		t.Fatalf("get session by hash: %v", err)
	}
	if gotSession.ID != session.ID || gotSession.UserID != user.ID {
		t.Fatalf("unexpected session loaded: %+v", gotSession)
	}

	if err := userSessionRepo.DeleteByUser(ctx, user.ID); err != nil {
		t.Fatalf("delete sessions by user: %v", err)
	}
	if _, err := userSessionRepo.GetByHash(ctx, session.TokenHash); err != domain.ErrUserSessionNotFound {
		t.Fatalf("expected session not found after delete, got %v", err)
	}
}

func TestTenantUserRepositoryListsTenantMembersAndUpdatesRole(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	userRepo := NewUserRepository(db)
	tenantRepo := NewTenantRepository(db)
	tenantUserRepo := NewTenantUserRepository(db)

	now := time.Now().UTC()
	tenant := &domain.Tenant{
		ID:        uuid.NewString(),
		Name:      "Tenant Team",
		Email:     "team@example.com",
		Active:    true,
		CreatedAt: now,
	}
	if err := tenantRepo.Create(ctx, tenant); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	owner := &domain.User{
		ID:           uuid.NewString(),
		Email:        "owner.team@example.com",
		Name:         "Owner Team",
		PasswordHash: "hash-owner",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := userRepo.Create(ctx, owner); err != nil {
		t.Fatalf("create owner user: %v", err)
	}

	member := &domain.User{
		ID:           uuid.NewString(),
		Email:        "member.team@example.com",
		Name:         "Member Team",
		PasswordHash: "hash-member",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := userRepo.Create(ctx, member); err != nil {
		t.Fatalf("create member user: %v", err)
	}

	ownerMembership := &domain.TenantUser{
		ID:        uuid.NewString(),
		TenantID:  tenant.ID,
		UserID:    owner.ID,
		Role:      domain.UserRoleOwner,
		CreatedAt: now,
	}
	if err := tenantUserRepo.Create(ctx, ownerMembership); err != nil {
		t.Fatalf("create owner membership: %v", err)
	}

	memberMembership := &domain.TenantUser{
		ID:        uuid.NewString(),
		TenantID:  tenant.ID,
		UserID:    member.ID,
		Role:      domain.UserRoleViewer,
		CreatedAt: now,
	}
	if err := tenantUserRepo.Create(ctx, memberMembership); err != nil {
		t.Fatalf("create member membership: %v", err)
	}

	members, err := tenantUserRepo.ListByTenant(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("list tenant members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 tenant members, got %d", len(members))
	}

	if err := tenantUserRepo.UpdateRole(ctx, memberMembership.ID, domain.UserRoleOperator); err != nil {
		t.Fatalf("update tenant role: %v", err)
	}

	updated, err := tenantUserRepo.GetByID(ctx, memberMembership.ID)
	if err != nil {
		t.Fatalf("get membership by id: %v", err)
	}
	if updated.Role != domain.UserRoleOperator {
		t.Fatalf("expected operator role, got %s", updated.Role)
	}
}
