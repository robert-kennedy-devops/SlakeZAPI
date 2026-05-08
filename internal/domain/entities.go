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
	PlanTrial   PlanName = "trial"
	PlanStarter PlanName = "starter"
	PlanGrowth  PlanName = "growth"
	PlanPro     PlanName = "pro"
)

type Plan struct {
	ID             string   `json:"id"`
	Name           PlanName `json:"name"`
	MonthlyLimit   int64    `json:"monthly_limit"`   // max messages per month
	PriceUSDCents  int64    `json:"price_usd_cents"` // platform price in cents
	WebhookEnabled bool     `json:"webhook_enabled"`
}

// DefaultPlans returns all available plans.
func DefaultPlans() []Plan {
	return []Plan{
		{ID: "plan_trial", Name: PlanTrial, MonthlyLimit: 0, PriceUSDCents: 0, WebhookEnabled: true},
		{ID: "plan_starter", Name: PlanStarter, MonthlyLimit: 3_000, PriceUSDCents: 7900, WebhookEnabled: false},
		{ID: "plan_growth", Name: PlanGrowth, MonthlyLimit: 15_000, PriceUSDCents: 14900, WebhookEnabled: true},
		{ID: "plan_pro", Name: PlanPro, MonthlyLimit: 60_000, PriceUSDCents: 29900, WebhookEnabled: true},
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

func PaidPlans() []Plan {
	plans := make([]Plan, 0, 3)
	for _, plan := range DefaultPlans() {
		if plan.Name == PlanTrial {
			continue
		}
		plans = append(plans, plan)
	}
	return plans
}

// ─── Subscription ───────────────────────────────────────────────────────────

type Subscription struct {
	ID                     string     `json:"id"`
	TenantID               string     `json:"tenant_id"`
	PlanID                 string     `json:"plan_id"`
	Plan                   *Plan      `json:"plan,omitempty"`
	Status                 string     `json:"status"` // trial, pending, active, past_due, cancelled
	Provider               string     `json:"provider,omitempty"`
	ProviderCustomerID     string     `json:"provider_customer_id,omitempty"`
	ProviderSubscriptionID string     `json:"provider_subscription_id,omitempty"`
	ProviderPriceID        string     `json:"provider_price_id,omitempty"`
	CurrentPeriodStart     *time.Time `json:"current_period_start,omitempty"`
	PeriodEnd              time.Time  `json:"period_end"`
	TrialEndsAt            *time.Time `json:"trial_ends_at,omitempty"`
	CancelAtPeriodEnd      bool       `json:"cancel_at_period_end"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (s *Subscription) TrialActive(now time.Time) bool {
	return s != nil && s.Status == "trial" && s.PeriodEnd.After(now)
}

// ─── Usage ──────────────────────────────────────────────────────────────────

type Usage struct {
	TenantID  string    `json:"tenant_id"`
	Month     string    `json:"month"` // format: "2024-04"
	Sent      int64     `json:"sent"`
	Received  int64     `json:"received"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─── Instances ──────────────────────────────────────────────────────────────

type Instance struct {
	ID        string        `json:"id"`
	TenantID  string        `json:"tenant_id"`
	Name      string        `json:"name"`
	Phone     string        `json:"phone,omitempty"`
	Status    SessionStatus `json:"status"`
	IsDefault bool          `json:"is_default"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
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

type ConversationState string

const (
	ConversationStateOpen     ConversationState = "open"
	ConversationStatePending  ConversationState = "pending"
	ConversationStateResolved ConversationState = "resolved"
)

type Message struct {
	ID            string        `json:"id"`
	TenantID      string        `json:"tenant_id"`
	InstanceID    string        `json:"instance_id,omitempty"`
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

type Conversation struct {
	ID              string            `json:"id"`
	TenantID        string            `json:"tenant_id"`
	InstanceID      string            `json:"instance_id"`
	Phone           string            `json:"phone"`
	LastMessageID   string            `json:"last_message_id,omitempty"`
	LastMessageBody string            `json:"last_message_body,omitempty"`
	LastDirection   string            `json:"last_direction,omitempty"`
	LastAt          time.Time         `json:"last_at"`
	State           ConversationState `json:"state"`
	AssignedUserID  string            `json:"assigned_user_id,omitempty"`
	AssignedName    string            `json:"assigned_name,omitempty"`
	Note            string            `json:"note,omitempty"`
	UnreadCount     int               `json:"unread_count"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type QueueJobView struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Attempt   int       `json:"attempt"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type QueueSnapshot struct {
	Jobs         int            `json:"jobs"`
	DeadLetters  int            `json:"dead_letters"`
	Workers      int            `json:"workers"`
	Recent       []QueueJobView `json:"recent"`
	DeadLettered []QueueJobView `json:"dead_lettered"`
}

type QueueRequeueResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type Group struct {
	JID              string             `json:"jid"`
	Name             string             `json:"name"`
	Topic            string             `json:"topic,omitempty"`
	ParticipantCount int                `json:"participant_count"`
	IsAnnounce       bool               `json:"is_announce"`
	IsLocked         bool               `json:"is_locked"`
	CreatedAt        time.Time          `json:"created_at"`
	Participants     []GroupParticipant `json:"participants,omitempty"`
}

type GroupParticipant struct {
	JID          string `json:"jid"`
	Phone        string `json:"phone"`
	IsAdmin      bool   `json:"is_admin"`
	IsSuperAdmin bool   `json:"is_super_admin"`
}

type CreateGroupRequest struct {
	InstanceID   string   `json:"instance_id,omitempty"`
	Name         string   `json:"name"`
	Participants []string `json:"participants"`
}

type UpdateGroupParticipantsRequest struct {
	InstanceID   string   `json:"instance_id,omitempty"`
	GroupJID     string   `json:"group_jid"`
	Participants []string `json:"participants"`
	Action       string   `json:"action"` // add, remove, promote, demote
}

type UpdateGroupInfoRequest struct {
	InstanceID  string `json:"instance_id,omitempty"`
	GroupJID    string `json:"group_jid"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type GroupInviteLink struct {
	GroupJID   string `json:"group_jid"`
	InviteLink string `json:"invite_link"`
}

type LeaveGroupRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	GroupJID   string `json:"group_jid"`
}

type InstanceProfile struct {
	InstanceID  string `json:"instance_id"`
	Phone       string `json:"phone,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	PictureURL  string `json:"picture_url,omitempty"`
}

type UpdateProfileRequest struct {
	InstanceID  string `json:"instance_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	PictureURL  string `json:"picture_url,omitempty"`
}

type PrivacySettings struct {
	LastSeen     string `json:"last_seen"`     // all, contacts, nobody
	ProfilePhoto string `json:"profile_photo"` // all, contacts, nobody
	Status       string `json:"status"`        // all, contacts, nobody
	ReadReceipts bool   `json:"read_receipts"`
	GroupAdd     string `json:"group_add"` // all, contacts, admins
}

type UpdatePrivacyRequest struct {
	InstanceID   string `json:"instance_id,omitempty"`
	LastSeen     string `json:"last_seen,omitempty"`
	ProfilePhoto string `json:"profile_photo,omitempty"`
	Status       string `json:"status,omitempty"`
	ReadReceipts *bool  `json:"read_receipts,omitempty"`
	GroupAdd     string `json:"group_add,omitempty"`
}

type BlockContactRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
}

type ContactAvatar struct {
	Phone string `json:"phone"`
	URL   string `json:"url,omitempty"`
}

type InstanceConfigRequest struct {
	InstanceID        string `json:"instance_id,omitempty"`
	AutoReadMessages  *bool  `json:"auto_read_messages,omitempty"`
	AutoReadStatus    *bool  `json:"auto_read_status,omitempty"`
	AutoRejectCalls   *bool  `json:"auto_reject_calls,omitempty"`
	RejectCallMessage string `json:"reject_call_message,omitempty"`
}

type WAContact struct {
	Phone        string `json:"phone"`
	FullName     string `json:"full_name,omitempty"`
	PushName     string `json:"push_name,omitempty"`
	BusinessName string `json:"business_name,omitempty"`
}

// ─── Webhook ─────────────────────────────────────────────────────────────────

type Webhook struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	InstanceID string    `json:"instance_id,omitempty"`
	URL        string    `json:"url"`
	Events     []string  `json:"events"` // e.g. ["message.received", "message.sent"]
	Secret     string    `json:"-"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
}

type WebhookDeliveryStatus string

const (
	WebhookDeliveryQueued    WebhookDeliveryStatus = "queued"
	WebhookDeliveryRetrying  WebhookDeliveryStatus = "retrying"
	WebhookDeliveryDelivered WebhookDeliveryStatus = "delivered"
	WebhookDeliveryFailed    WebhookDeliveryStatus = "failed"
	WebhookDeliveryReplayed  WebhookDeliveryStatus = "replayed"
)

type WebhookDelivery struct {
	ID             string                `json:"id"`
	WebhookID      string                `json:"webhook_id"`
	TenantID       string                `json:"tenant_id"`
	InstanceID     string                `json:"instance_id,omitempty"`
	EventType      EventType             `json:"event_type"`
	WebhookURL     string                `json:"webhook_url"`
	Status         WebhookDeliveryStatus `json:"status"`
	Attempts       int                   `json:"attempts"`
	ResponseStatus int                   `json:"response_status,omitempty"`
	ResponseBody   string                `json:"response_body,omitempty"`
	LastError      string                `json:"last_error,omitempty"`
	PayloadJSON    []byte                `json:"payload_json,omitempty"`
	DeliveredAt    *time.Time            `json:"delivered_at,omitempty"`
	LastAttemptAt  *time.Time            `json:"last_attempt_at,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type AuditLog struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	InstanceID string                 `json:"instance_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ─── WhatsApp Session ────────────────────────────────────────────────────────

type SessionStatus string

const (
	SessionStatusDisconnected SessionStatus = "disconnected"
	SessionStatusConnecting   SessionStatus = "connecting"
	SessionStatusConnected    SessionStatus = "connected"
)

type Session struct {
	TenantID   string        `json:"tenant_id"`
	InstanceID string        `json:"instance_id"`
	DeviceJID  string        `json:"-"`
	Status     SessionStatus `json:"status"`
	Phone      string        `json:"phone"`
	UpdatedAt  time.Time     `json:"updated_at"`
	LastEvent  string        `json:"last_event,omitempty"`
	LastError  string        `json:"last_error,omitempty"`
	QRCode     string        `json:"qr_code,omitempty"`
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
	Type       EventType   `json:"type"`
	TenantID   string      `json:"tenant_id"`
	InstanceID string      `json:"instance_id,omitempty"`
	Payload    interface{} `json:"payload"`
}

type WebhookEnvelope struct {
	ID         string      `json:"id"`
	Version    string      `json:"version"`
	Type       EventType   `json:"type"`
	TenantID   string      `json:"tenant_id"`
	InstanceID string      `json:"instance_id,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
	Payload    interface{} `json:"payload"`
}

// ─── Request/Response DTOs ───────────────────────────────────────────────────

type SendMessageRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
	Message    string `json:"message"`
}

type InteractiveMessageButton struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type InteractiveMessageListRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type InteractiveMessageListSection struct {
	Title string                      `json:"title"`
	Rows  []InteractiveMessageListRow `json:"rows"`
}

type InteractiveMessageRequest struct {
	InstanceID string                          `json:"instance_id,omitempty"`
	Phone      string                          `json:"phone"`
	Type       string                          `json:"type"`
	Header     string                          `json:"header,omitempty"`
	Body       string                          `json:"body"`
	Footer     string                          `json:"footer,omitempty"`
	Buttons    []InteractiveMessageButton      `json:"buttons,omitempty"`
	ButtonText string                          `json:"button_text,omitempty"`
	Sections   []InteractiveMessageListSection `json:"sections,omitempty"`
	Options    []string                        `json:"options,omitempty"`
	MaxSelect  int                             `json:"max_select,omitempty"`
}

type GroupMessageRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	GroupJID   string `json:"group_jid"`
	Message    string `json:"message"`
}

type LocationMessageRequest struct {
	InstanceID string  `json:"instance_id,omitempty"`
	Phone      string  `json:"phone"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Name       string  `json:"name,omitempty"`
	Address    string  `json:"address,omitempty"`
}

type ContactCardRequest struct {
	InstanceID string   `json:"instance_id,omitempty"`
	Phone      string   `json:"phone"`
	Contacts   []string `json:"contacts"`
}

type ReactMessageRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
	MessageID  string `json:"message_id"`
	Emoji      string `json:"emoji"`
}

type DeleteMessageRequest struct {
	InstanceID  string `json:"instance_id,omitempty"`
	Phone       string `json:"phone"`
	MessageID   string `json:"message_id"`
	ForEveryone bool   `json:"for_everyone"`
}

type QuotedSendRequest struct {
	InstanceID      string `json:"instance_id,omitempty"`
	Phone           string `json:"phone"`
	Message         string `json:"message"`
	QuotedMessageID string `json:"quoted_message_id"`
}

type StatusMessageRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Type       string `json:"type"`
	Message    string `json:"message,omitempty"`
	URL        string `json:"url,omitempty"`
	Caption    string `json:"caption,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
	Data       []byte `json:"-"`
}

type UpdateConversationRequest struct {
	State          ConversationState `json:"state,omitempty"`
	AssignedUserID string            `json:"assigned_user_id,omitempty"`
	Note           string            `json:"note,omitempty"`
}

type ResolveContactsRequest struct {
	InstanceID string   `json:"instance_id,omitempty"`
	Phones     []string `json:"phones"`
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
	InstanceID string   `json:"instance_id,omitempty"`
	Phones     []string `json:"phones"`
	Message    string   `json:"message"`
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
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	Caption    string `json:"caption,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
	Data       []byte `json:"-"`
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

type MessageTranscript struct {
	Text     string `json:"text"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
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
	Tenant    *Tenant       `json:"tenant"`
	Session   *Session      `json:"session,omitempty"`
	Instances []Instance    `json:"instances,omitempty"`
	Usage     *Usage        `json:"usage,omitempty"`
	Plan      *Subscription `json:"plan,omitempty"`
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
	InstanceID string `json:"instance_id,omitempty"`
	QRCode     string `json:"qr_code,omitempty"` // raw QR payload from WhatsApp
	QRPNGURL   string `json:"qr_png_url,omitempty"`
	QRPageURL  string `json:"qr_page_url,omitempty"`
	Status     string `json:"status"`
	Phone      string `json:"phone,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}

type RegisterWebhookRequest struct {
	InstanceID string   `json:"instance_id,omitempty"`
	URL        string   `json:"url"`
	Events     []string `json:"events"`
}

type ReplayWebhookDeliveryResponse struct {
	DeliveryID string                `json:"delivery_id"`
	Status     WebhookDeliveryStatus `json:"status"`
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

type CheckoutSessionRequest struct {
	TenantID      string
	TenantName    string
	CustomerEmail string
	Plan          Plan
	SuccessURL    string
	CancelURL     string
}

type CheckoutSessionResponse struct {
	URL       string `json:"url"`
	SessionID string `json:"session_id"`
	Provider  string `json:"provider"`
}

type BillingPortalResponse struct {
	URL      string `json:"url"`
	Provider string `json:"provider"`
}

type CreateCheckoutRequest struct {
	Plan PlanName `json:"plan"`
}

type ChangeSubscriptionPlanRequest struct {
	Plan PlanName `json:"plan"`
}

type BillingActionResponse struct {
	Subscription     *Subscription `json:"subscription,omitempty"`
	CheckoutURL      string        `json:"checkout_url,omitempty"`
	PortalURL        string        `json:"portal_url,omitempty"`
	Provider         string        `json:"provider,omitempty"`
	RequiresCheckout bool          `json:"requires_checkout,omitempty"`
	Message          string        `json:"message,omitempty"`
}

type BillingWebhookEvent struct {
	EventType              string
	TenantID               string
	PlanID                 string
	Status                 string
	Provider               string
	ProviderCustomerID     string
	ProviderSubscriptionID string
	ProviderPriceID        string
	CurrentPeriodStart     *time.Time
	PeriodEnd              *time.Time
	CancelAtPeriodEnd      bool
}

// ─── Campaigns ──────────────────────────────────────────────────────────────

type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusScheduled CampaignStatus = "scheduled"
	CampaignStatusRunning   CampaignStatus = "running"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusFailed    CampaignStatus = "failed"
	CampaignStatusCancelled CampaignStatus = "cancelled"
)

type Campaign struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	InstanceID     string         `json:"instance_id"`
	Name           string         `json:"name"`
	Message        string         `json:"message"`
	Status         CampaignStatus `json:"status"`
	ScheduledAt    *time.Time     `json:"scheduled_at,omitempty"`
	LastExecutedAt *time.Time     `json:"last_executed_at,omitempty"`
	TotalContacts  int            `json:"total_contacts"`
	SentCount      int            `json:"sent_count"`
	FailedCount    int            `json:"failed_count"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type CampaignRecipient struct {
	ID         string        `json:"id"`
	CampaignID string        `json:"campaign_id"`
	InputPhone string        `json:"input_phone"`
	Phone      string        `json:"phone,omitempty"`
	Name       string        `json:"name,omitempty"`
	Variables  string        `json:"variables,omitempty"`
	MessageID  string        `json:"message_id,omitempty"`
	Status     MessageStatus `json:"status"`
	Error      string        `json:"error,omitempty"`
	IsWhatsApp bool          `json:"is_whatsapp"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type CreateInstanceRequest struct {
	Name string `json:"name"`
}

type CreateCampaignRecipient struct {
	Phone     string            `json:"phone"`
	Name      string            `json:"name,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

type CreateCampaignRequest struct {
	InstanceID  string                    `json:"instance_id,omitempty"`
	Name        string                    `json:"name"`
	Message     string                    `json:"message"`
	ScheduledAt *time.Time                `json:"scheduled_at,omitempty"`
	Recipients  []CreateCampaignRecipient `json:"recipients"`
}

type UpdateCampaignStatusRequest struct {
	Status CampaignStatus `json:"status"`
}

// ─── Chat Operations ─────────────────────────────────────────────────────────

type ArchiveChatRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
	Archive    bool   `json:"archive"`
}

type MuteChatRequest struct {
	InstanceID    string `json:"instance_id,omitempty"`
	Phone         string `json:"phone"`
	Mute          bool   `json:"mute"`
	DurationHours int    `json:"duration_hours,omitempty"` // 0 = forever when muting
}

type PinChatRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
	Pin        bool   `json:"pin"`
}

type MarkChatReadRequest struct {
	InstanceID    string `json:"instance_id,omitempty"`
	Phone         string `json:"phone"`
	Read          bool   `json:"read"`
	LastMessageID string `json:"last_message_id,omitempty"`
}

// ─── Message Operations ──────────────────────────────────────────────────────

type EditMessageRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
	MessageID  string `json:"message_id"`
	NewMessage string `json:"new_message"`
}

type ForwardMessageRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
	Message    string `json:"message"`
}

type StarMessageRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
	MessageID  string `json:"message_id"`
	Starred    bool   `json:"starred"`
	FromMe     bool   `json:"from_me"`
}

// ─── Instance Operations ─────────────────────────────────────────────────────

type PairPhoneRequest struct {
	InstanceID string `json:"instance_id,omitempty"`
	Phone      string `json:"phone"`
}

type PairPhoneResponse struct {
	Code string `json:"code"`
}
