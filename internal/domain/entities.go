package domain

import (
	"time"
)

// ─── Tenant ────────────────────────────────────────────────────────────────

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

// ─── Users ──────────────────────────────────────────────────────────────────

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserRole string

const (
	UserRoleOwner    UserRole = "owner"
	UserRoleAdmin    UserRole = "admin"
	UserRoleOperator UserRole = "operator"
	UserRoleViewer   UserRole = "viewer"
)

type TenantUser struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type TenantMember struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      UserRole  `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type UserSession struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	TokenHash        string    `json:"-"`
	RefreshTokenHash string    `json:"-"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	CreatedAt        time.Time `json:"created_at"`
	LastUsedAt       time.Time `json:"last_used_at"`
}

// ─── APIKey ─────────────────────────────────────────────────────────────────

type APIKey struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	KeyHash   string    `json:"-"` // stored hashed, never returned
	KeyPrefix string    `json:"key_prefix"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	Active    bool      `json:"active"`
}

// ─── Plan ───────────────────────────────────────────────────────────────────

type PlanName string

const (
	PlanStarter PlanName = "starter"
	PlanGrowth  PlanName = "growth"
	PlanPro     PlanName = "pro"
)

type Plan struct {
	ID             string   `json:"id"`
	Name           PlanName `json:"name"`
	MonthlyLimit   int64    `json:"monthly_limit"`   // max messages per month
	PriceUSDCents  int64    `json:"price_usd_cents"` // price in cents
	WebhookEnabled bool     `json:"webhook_enabled"`
}

// DefaultPlans returns all available plans.
func DefaultPlans() []Plan {
	return []Plan{
		{ID: "plan_starter", Name: PlanStarter, MonthlyLimit: 1_000, PriceUSDCents: 0, WebhookEnabled: false},
		{ID: "plan_growth", Name: PlanGrowth, MonthlyLimit: 10_000, PriceUSDCents: 2900, WebhookEnabled: true},
		{ID: "plan_pro", Name: PlanPro, MonthlyLimit: 100_000, PriceUSDCents: 9900, WebhookEnabled: true},
	}
}

func PlanByName(name PlanName) (Plan, bool) {
	for _, plan := range DefaultPlans() {
		if plan.Name == name {
			return plan, true
		}
	}
	return Plan{}, false
}

// ─── Subscription ───────────────────────────────────────────────────────────

type Subscription struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	PlanID    string    `json:"plan_id"`
	Plan      *Plan     `json:"plan,omitempty"`
	Status    string    `json:"status"` // active, cancelled, past_due
	PeriodEnd time.Time `json:"period_end"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── Usage ──────────────────────────────────────────────────────────────────

type Usage struct {
	TenantID  string    `json:"tenant_id"`
	Month     string    `json:"month"` // format: "2024-04"
	Sent      int64     `json:"sent"`
	Received  int64     `json:"received"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─── Message ─────────────────────────────────────────────────────────────────

type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

type Message struct {
	ID            string        `json:"id"`
	TenantID      string        `json:"tenant_id"`
	WhatsAppID    string        `json:"whatsapp_id"`
	Phone         string        `json:"phone"`
	Body          string        `json:"body"`
	Type          string        `json:"type"`
	MimeType      string        `json:"mime_type,omitempty"`
	FileName      string        `json:"file_name,omitempty"`
	MediaURL      string        `json:"media_url,omitempty"`
	DirectPath    string        `json:"direct_path,omitempty"`
	FileLength    int64         `json:"file_length,omitempty"`
	MediaKey      []byte        `json:"-"`
	FileSHA256    []byte        `json:"-"`
	FileEncSHA256 []byte        `json:"-"`
	Direction     string        `json:"direction"` // inbound / outbound
	Status        MessageStatus `json:"status"`
	SentAt        time.Time     `json:"sent_at"`
	CreatedAt     time.Time     `json:"created_at"`
}

// ─── Webhook ─────────────────────────────────────────────────────────────────

type Webhook struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"` // e.g. ["message.received", "message.sent"]
	Secret    string    `json:"-"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── WhatsApp Session ────────────────────────────────────────────────────────

type SessionStatus string

const (
	SessionStatusDisconnected SessionStatus = "disconnected"
	SessionStatusConnecting   SessionStatus = "connecting"
	SessionStatusConnected    SessionStatus = "connected"
)

type Session struct {
	TenantID  string        `json:"tenant_id"`
	DeviceJID string        `json:"-"`
	Status    SessionStatus `json:"status"`
	Phone     string        `json:"phone"`
	UpdatedAt time.Time     `json:"updated_at"`
	LastEvent string        `json:"last_event,omitempty"`
	LastError string        `json:"last_error,omitempty"`
	QRCode    string        `json:"qr_code,omitempty"`
}

// ─── Event ───────────────────────────────────────────────────────────────────

type EventType string

const (
	EventMessageReceived  EventType = "message.received"
	EventMessageSent      EventType = "message.sent"
	EventMessageStatus    EventType = "message.status"
	EventConnectionUpdate EventType = "connection.update"
)

type Event struct {
	Type     EventType   `json:"type"`
	TenantID string      `json:"tenant_id"`
	Payload  interface{} `json:"payload"`
}

type WebhookEnvelope struct {
	ID        string      `json:"id"`
	Version   string      `json:"version"`
	Type      EventType   `json:"type"`
	TenantID  string      `json:"tenant_id"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// ─── Request/Response DTOs ───────────────────────────────────────────────────

type SendMessageRequest struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

type ResolveContactsRequest struct {
	Phones []string `json:"phones"`
}

type ResolvedContact struct {
	InputPhone   string `json:"input_phone"`
	LookupPhone  string `json:"lookup_phone"`
	Phone        string `json:"phone"`
	JID          string `json:"jid,omitempty"`
	IsWhatsApp   bool   `json:"is_whatsapp"`
	VerifiedName string `json:"verified_name,omitempty"`
	Error        string `json:"error,omitempty"`
}

type BulkSendMessageRequest struct {
	Phones  []string `json:"phones"`
	Message string   `json:"message"`
}

type BulkSendMessageItem struct {
	InputPhone string        `json:"input_phone"`
	Phone      string        `json:"phone,omitempty"`
	IsWhatsApp bool          `json:"is_whatsapp"`
	MessageID  string        `json:"message_id,omitempty"`
	Status     MessageStatus `json:"status,omitempty"`
	Error      string        `json:"error,omitempty"`
}

type BulkSendMessageResponse struct {
	Total    int                   `json:"total"`
	Accepted int                   `json:"accepted"`
	Sent     int                   `json:"sent"`
	Failed   int                   `json:"failed"`
	Results  []BulkSendMessageItem `json:"results"`
}

type SendMediaMessageRequest struct {
	Phone    string `json:"phone"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Caption  string `json:"caption,omitempty"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Data     []byte `json:"-"`
}

type SendMessageResponse struct {
	MessageID string        `json:"message_id"`
	Status    MessageStatus `json:"status"`
}

type MediaDownload struct {
	FileName string
	MimeType string
	Data     []byte
}

type CreateAPIKeyRequest struct {
	Label string `json:"label"`
}

type CreateAPIKeyResponse struct {
	KeyID  string `json:"key_id"`
	APIKey string `json:"api_key"` // only returned once, on creation
	Prefix string `json:"prefix"`
}

type TenantSummary struct {
	Tenant  *Tenant       `json:"tenant"`
	Session *Session      `json:"session,omitempty"`
	Usage   *Usage        `json:"usage,omitempty"`
	Plan    *Subscription `json:"plan,omitempty"`
}

type BootstrapTenantRequest struct {
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Plan  PlanName `json:"plan"`
}

type BootstrapTenantResponse struct {
	TenantID string                `json:"tenant_id"`
	Plan     PlanName              `json:"plan"`
	APIKey   *CreateAPIKeyResponse `json:"api_key"`
}

type ConnectRequest struct {
	// future: phone number hint
}

type ConnectResponse struct {
	QRCode    string `json:"qr_code,omitempty"` // raw QR payload from WhatsApp
	QRPNGURL  string `json:"qr_png_url,omitempty"`
	QRPageURL string `json:"qr_page_url,omitempty"`
	Status    string `json:"status"`
	Phone     string `json:"phone,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

type RegisterWebhookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

type SignUpRequest struct {
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Password   string   `json:"password"`
	TenantName string   `json:"tenant_name"`
	Plan       PlanName `json:"plan"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthSessionResponse struct {
	Token            string      `json:"token"`
	ExpiresAt        time.Time   `json:"expires_at"`
	RefreshExpiresAt time.Time   `json:"refresh_expires_at"`
	User             *User       `json:"user"`
	Tenant           *Tenant     `json:"tenant"`
	Membership       *TenantUser `json:"membership"`
}

type CurrentUserResponse struct {
	User        *User        `json:"user"`
	Tenant      *Tenant      `json:"tenant"`
	Membership  *TenantUser  `json:"membership"`
	Memberships []TenantUser `json:"memberships,omitempty"`
}

type AddTenantMemberRequest struct {
	Email string   `json:"email"`
	Role  UserRole `json:"role"`
}

type UpdateTenantMemberRoleRequest struct {
	Role UserRole `json:"role"`
}
