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
