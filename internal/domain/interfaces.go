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

type InstanceRepository interface {
	Create(ctx context.Context, instance *Instance) error
	GetByID(ctx context.Context, id string) (*Instance, error)
	GetDefaultByTenant(ctx context.Context, tenantID string) (*Instance, error)
	ListByTenant(ctx context.Context, tenantID string) ([]Instance, error)
	UpdateStatus(ctx context.Context, instanceID string, status SessionStatus, phone string, updatedAt time.Time) error
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
	GetByWhatsAppID(ctx context.Context, tenantID, instanceID, whatsappID string) (*Message, error)
	ListByTenant(ctx context.Context, tenantID, instanceID string, limit, offset int) ([]Message, error)
	ListConversations(ctx context.Context, tenantID, instanceID string, limit, offset int) ([]Conversation, error)
	UpdateConversation(ctx context.Context, convo *Conversation) error
	GetConversation(ctx context.Context, tenantID, instanceID, phone string) (*Conversation, error)
	UpdateStatus(ctx context.Context, id string, status MessageStatus) error
}

// WebhookRepository handles webhook persistence.
type WebhookRepository interface {
	Create(ctx context.Context, wh *Webhook) error
	GetByTenant(ctx context.Context, tenantID, instanceID string) ([]Webhook, error)
	GetByID(ctx context.Context, id string) (*Webhook, error)
	Delete(ctx context.Context, id string) error
	CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error
	UpdateDeliveryAttempt(ctx context.Context, id string, status WebhookDeliveryStatus, responseStatus int, responseBody, lastError string, deliveredAt, attemptedAt *time.Time) error
	ListDeliveries(ctx context.Context, tenantID, instanceID, webhookID string, limit int) ([]WebhookDelivery, error)
	GetDeliveryByID(ctx context.Context, id string) (*WebhookDelivery, error)
}

// SessionRepository handles persistence of WhatsApp session metadata.
type SessionRepository interface {
	Upsert(ctx context.Context, session *Session) error
	GetByInstance(ctx context.Context, instanceID string) (*Session, error)
	ListSessions(ctx context.Context) ([]Session, error)
	Delete(ctx context.Context, instanceID string) error
}

type CampaignRepository interface {
	Create(ctx context.Context, campaign *Campaign, recipients []CampaignRecipient) error
	GetByID(ctx context.Context, id string) (*Campaign, error)
	ListByTenant(ctx context.Context, tenantID, instanceID string) ([]Campaign, error)
	ListRecipients(ctx context.Context, campaignID string) ([]CampaignRecipient, error)
	ListDue(ctx context.Context, now time.Time) ([]Campaign, error)
	UpdateStatus(ctx context.Context, id string, status CampaignStatus, lastExecutedAt *time.Time) error
	UpdateRecipientResult(ctx context.Context, recipientID, phone, messageID string, status MessageStatus, isWhatsApp bool, errText string) error
	UpdateCounters(ctx context.Context, campaignID string, sentCount, failedCount int) error
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
	Connect(ctx context.Context, tenantID, instanceID string) (string, error) // returns QR code
	Disconnect(ctx context.Context, tenantID, instanceID string) error
	Logout(ctx context.Context, tenantID, instanceID string) error
	SendMessage(ctx context.Context, tenantID, instanceID, phone, message string) (string, error) // returns whatsapp msg ID
	SendMediaMessage(ctx context.Context, tenantID, instanceID string, req SendMediaMessageRequest) (string, error)
	SendInteractiveMessage(ctx context.Context, tenantID, instanceID string, req InteractiveMessageRequest) (string, error)
	SendGroupMessage(ctx context.Context, tenantID, instanceID string, req GroupMessageRequest) (string, error)
	PostStatus(ctx context.Context, tenantID, instanceID string, req StatusMessageRequest) (string, error)
	ListGroups(ctx context.Context, tenantID, instanceID string) ([]Group, error)
	ListContacts(ctx context.Context, tenantID, instanceID string) ([]WAContact, error)
	ResolveContacts(ctx context.Context, tenantID, instanceID string, phones []string) ([]ResolvedContact, error)
	DownloadMedia(ctx context.Context, tenantID, instanceID string, msg *Message) (*MediaDownload, error)
	GetSession(ctx context.Context, tenantID, instanceID string) (*Session, error)
	GetStatus(ctx context.Context, tenantID, instanceID string) SessionStatus
}

// EventBus broadcasts events internally (to WebSocket clients, webhooks, etc.)
type EventBus interface {
	Publish(event Event)
	Subscribe(tenantID string) (<-chan Event, func())
	SubscribeAll() (<-chan Event, func())
}

// WebhookDeliveryExecutor handles outbound webhook POST requests.
type WebhookDeliveryExecutor interface {
	Deliver(ctx context.Context, webhook Webhook, event Event) error
}

type WebhookReplayService interface {
	ReplayDelivery(ctx context.Context, wh Webhook, source *WebhookDelivery) (*WebhookDelivery, error)
}

// BillingService enforces plan limits and tracks usage.
type BillingService interface {
	CheckLimit(ctx context.Context, tenantID string) error // returns ErrLimitExceeded if over quota
	TrackSent(ctx context.Context, tenantID string) error
	TrackReceived(ctx context.Context, tenantID string) error
	GetUsage(ctx context.Context, tenantID string) (*Usage, error)
}

type QueueService interface {
	Snapshot() QueueSnapshot
	DeadLetters(limit int) []QueueJobView
	RequeueDeadLetter(id string) error
}

type AuditLogRepository interface {
	Create(ctx context.Context, item *AuditLog) error
	List(ctx context.Context, tenantID, instanceID, actionPrefix string, limit int) ([]AuditLog, error)
}
