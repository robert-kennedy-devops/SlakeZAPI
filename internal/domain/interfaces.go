package domain

import (
	"context"
	"time"
)

// ─── Repository Interfaces ───────────────────────────────────────────────────

// TenantRepository handles tenant persistence.
type TenantRepository interface {
	Create(ctx context.Context, tenant *Tenant) error
	GetByID(ctx context.Context, id string) (*Tenant, error)
	GetByEmail(ctx context.Context, email string) (*Tenant, error)
	Update(ctx context.Context, tenant *Tenant) error
}

// UserRepository handles user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
}

// TenantUserRepository handles user membership in tenants.
type TenantUserRepository interface {
	Create(ctx context.Context, tenantUser *TenantUser) error
	GetByID(ctx context.Context, id string) (*TenantUser, error)
	GetByUserAndTenant(ctx context.Context, userID, tenantID string) (*TenantUser, error)
	ListByUser(ctx context.Context, userID string) ([]TenantUser, error)
	ListByTenant(ctx context.Context, tenantID string) ([]TenantMember, error)
	UpdateRole(ctx context.Context, id string, role UserRole) error
}

// UserSessionRepository handles authenticated user sessions.
type UserSessionRepository interface {
	Create(ctx context.Context, session *UserSession) error
	GetByHash(ctx context.Context, tokenHash string) (*UserSession, error)
	GetByRefreshHash(ctx context.Context, refreshHash string) (*UserSession, error)
	DeleteByID(ctx context.Context, id string) error
	DeleteByUser(ctx context.Context, userID string) error
	Touch(ctx context.Context, id string, accessExpiresAt time.Time) error
	RotateTokens(ctx context.Context, id, accessHash, refreshHash string, accessExpiresAt, refreshExpiresAt time.Time) error
}

// APIKeyRepository handles API key persistence.
type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByHash(ctx context.Context, hash string) (*APIKey, error)
	ListByTenant(ctx context.Context, tenantID string) ([]APIKey, error)
	GetByID(ctx context.Context, keyID string) (*APIKey, error)
	UpdateLastUsed(ctx context.Context, keyID string) error
	Revoke(ctx context.Context, keyID string) error
}

// MessageRepository handles message persistence.
type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	GetByID(ctx context.Context, id string) (*Message, error)
	GetByWhatsAppID(ctx context.Context, tenantID, whatsappID string) (*Message, error)
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]Message, error)
	UpdateStatus(ctx context.Context, id string, status MessageStatus) error
}

// WebhookRepository handles webhook persistence.
type WebhookRepository interface {
	Create(ctx context.Context, wh *Webhook) error
	GetByTenant(ctx context.Context, tenantID string) ([]Webhook, error)
	Delete(ctx context.Context, id string) error
}

// SessionRepository handles persistence of WhatsApp session metadata.
type SessionRepository interface {
	Upsert(ctx context.Context, session *Session) error
	GetByTenant(ctx context.Context, tenantID string) (*Session, error)
	ListTenantIDs(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, tenantID string) error
}

// SubscriptionRepository handles subscription + billing persistence.
type SubscriptionRepository interface {
	GetByTenant(ctx context.Context, tenantID string) (*Subscription, error)
	Upsert(ctx context.Context, sub *Subscription) error
}

// UsageRepository handles message usage tracking.
type UsageRepository interface {
	IncrementSent(ctx context.Context, tenantID string) error
	IncrementReceived(ctx context.Context, tenantID string) error
	GetCurrentMonth(ctx context.Context, tenantID string) (*Usage, error)
}

// ─── Service Interfaces ──────────────────────────────────────────────────────

// WhatsAppService manages WhatsApp sessions and messaging.
type WhatsAppService interface {
	Connect(ctx context.Context, tenantID string) (string, error) // returns QR code
	Disconnect(ctx context.Context, tenantID string) error
	Logout(ctx context.Context, tenantID string) error
	SendMessage(ctx context.Context, tenantID, phone, message string) (string, error) // returns whatsapp msg ID
	SendMediaMessage(ctx context.Context, tenantID string, req SendMediaMessageRequest) (string, error)
	DownloadMedia(ctx context.Context, tenantID string, msg *Message) (*MediaDownload, error)
	GetSession(ctx context.Context, tenantID string) (*Session, error)
	GetStatus(ctx context.Context, tenantID string) SessionStatus
}

// EventBus broadcasts events internally (to WebSocket clients, webhooks, etc.)
type EventBus interface {
	Publish(event Event)
	Subscribe(tenantID string) (<-chan Event, func())
	SubscribeAll() (<-chan Event, func())
}

// WebhookDelivery handles outbound webhook POST requests.
type WebhookDelivery interface {
	Deliver(ctx context.Context, webhook Webhook, event Event) error
}

// BillingService enforces plan limits and tracks usage.
type BillingService interface {
	CheckLimit(ctx context.Context, tenantID string) error // returns ErrLimitExceeded if over quota
	TrackSent(ctx context.Context, tenantID string) error
	TrackReceived(ctx context.Context, tenantID string) error
	GetUsage(ctx context.Context, tenantID string) (*Usage, error)
}
