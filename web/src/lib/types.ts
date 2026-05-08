export type UserRole = "owner" | "admin" | "operator" | "viewer";

export interface User {
  id: string;
  email: string;
  name: string;
  active: boolean;
  created_at: string;
}

export interface Tenant {
  id: string;
  name: string;
  email: string;
  active: boolean;
  created_at: string;
}

export interface TenantUser {
  id: string;
  tenant_id: string;
  user_id: string;
  role: UserRole;
  created_at: string;
}

export interface UserSessionResponse {
  token: string;
  expires_at: string;
  refresh_expires_at: string;
  user: User;
  tenant: Tenant;
  membership: TenantUser;
}

export interface CurrentUserResponse {
  user: User;
  tenant: Tenant;
  membership: TenantUser;
  memberships: TenantUser[];
}

export interface TenantMember {
  id: string;
  tenant_id: string;
  user_id: string;
  email: string;
  name: string;
  role: UserRole;
  active: boolean;
  created_at: string;
}

export interface Usage {
  tenant_id: string;
  month: string;
  sent: number;
  received: number;
  updated_at: string;
}

export interface SessionStatus {
  tenant_id: string;
  instance_id: string;
  status: "disconnected" | "connecting" | "connected";
  phone?: string;
  updated_at?: string;
  last_event?: string;
  last_error?: string;
  qr_code?: string;
  qr_png_url?: string;
  qr_page_url?: string;
}

export interface Subscription {
  id: string;
  tenant_id: string;
  plan_id: string;
  plan?: {
    id: string;
    name: string;
    monthly_limit: number;
    price_usd_cents: number;
    webhook_enabled: boolean;
  };
  status: string;
  provider?: string;
  provider_customer_id?: string;
  provider_subscription_id?: string;
  provider_price_id?: string;
  current_period_start?: string;
  period_end: string;
  trial_ends_at?: string;
  cancel_at_period_end: boolean;
  created_at: string;
  updated_at?: string;
}

export interface TenantSummary {
  tenant: Tenant;
  session?: SessionStatus;
  instances?: Instance[];
  usage?: Usage;
  plan?: Subscription;
}

export interface BillingActionResponse {
  subscription?: Subscription;
  checkout_url?: string;
  portal_url?: string;
  provider?: string;
  requires_checkout?: boolean;
  message?: string;
}

export interface Instance {
  id: string;
  tenant_id: string;
  name: string;
  phone?: string;
  status: "disconnected" | "connecting" | "connected";
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: string;
  tenant_id: string;
  instance_id?: string;
  whatsapp_id: string;
  phone: string;
  body: string;
  type: string;
  mime_type?: string;
  file_name?: string;
  media_url?: string;
  direct_path?: string;
  file_length?: number;
  direction: "inbound" | "outbound";
  status: string;
  sent_at: string;
  created_at: string;
}

export interface MessageTranscript {
  text: string;
  provider: string;
  model: string;
}

export interface Conversation {
  id: string;
  tenant_id: string;
  instance_id: string;
  phone: string;
  last_message_id?: string;
  last_message_body?: string;
  last_direction?: string;
  last_at: string;
  state: "open" | "pending" | "resolved";
  assigned_user_id?: string;
  assigned_name?: string;
  note?: string;
  unread_count: number;
  created_at: string;
  updated_at: string;
}

export interface QueueJobView {
  id: string;
  kind: string;
  attempt: number;
  status: string;
  error?: string;
  created_at: string;
  updated_at: string;
}

export interface QueueSnapshot {
  jobs: number;
  dead_letters: number;
  workers: number;
  recent: QueueJobView[];
  dead_lettered: QueueJobView[];
}

export interface QueueRequeueResponse {
  job_id: string;
  status: string;
}

export interface GroupParticipant {
  jid: string;
  phone: string;
  is_admin: boolean;
  is_super_admin: boolean;
}

export interface Group {
  jid: string;
  name: string;
  topic?: string;
  participant_count: number;
  is_announce: boolean;
  is_locked: boolean;
  created_at: string;
  participants?: GroupParticipant[];
}

export interface GroupInviteLink {
  group_jid: string;
  invite_link: string;
}

export interface InstanceProfile {
  instance_id: string;
  phone?: string;
  name?: string;
  description?: string;
  picture_url?: string;
}

export interface PrivacySettings {
  last_seen: string;
  profile_photo: string;
  status: string;
  read_receipts: boolean;
  group_add: string;
}

export interface ContactAvatar {
  phone: string;
  url?: string;
}

export interface LocationMessageRequest {
  phone: string;
  latitude: number;
  longitude: number;
  name?: string;
  address?: string;
  instance_id?: string;
}

export interface ContactCardRequest {
  phone: string;
  contacts: string[];
  instance_id?: string;
}

export interface ReactMessageRequest {
  phone: string;
  message_id: string;
  emoji: string;
  instance_id?: string;
}

export interface DeleteMessageRequest {
  phone: string;
  message_id: string;
  for_everyone: boolean;
  instance_id?: string;
}

export interface QuotedSendRequest {
  phone: string;
  message: string;
  quoted_message_id: string;
  instance_id?: string;
}

export interface WAContact {
  phone: string;
  full_name?: string;
  push_name?: string;
  business_name?: string;
}

export interface Webhook {
  id: string;
  tenant_id: string;
  instance_id?: string;
  url: string;
  events: string[];
  active: boolean;
  created_at: string;
}

export interface WebhookDelivery {
  id: string;
  webhook_id: string;
  tenant_id: string;
  instance_id?: string;
  event_type: string;
  webhook_url: string;
  status: "queued" | "retrying" | "delivered" | "failed" | "replayed";
  attempts: number;
  response_status?: number;
  response_body?: string;
  last_error?: string;
  payload_json?: string;
  delivered_at?: string;
  last_attempt_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AuditLog {
  id: string;
  tenant_id: string;
  instance_id?: string;
  user_id?: string;
  request_id?: string;
  action: string;
  resource: string;
  payload?: Record<string, unknown>;
  created_at: string;
}

export interface APIKey {
  id: string;
  tenant_id: string;
  key_prefix: string;
  label: string;
  created_at: string;
  last_used: string;
  active: boolean;
}

export interface CreateAPIKeyResponse {
  key_id: string;
  api_key: string;
  prefix: string;
}

export interface SendMessageResponse {
  message_id: string;
  status: string;
}

export interface ResolvedContact {
  input_phone: string;
  lookup_phone: string;
  phone: string;
  jid?: string;
  is_whatsapp: boolean;
  verified_name?: string;
  error?: string;
}

export interface BulkSendMessageItem {
  input_phone: string;
  phone?: string;
  is_whatsapp: boolean;
  message_id?: string;
  status?: string;
  error?: string;
}

export interface BulkSendMessageResponse {
  total: number;
  accepted: number;
  sent: number;
  failed: number;
  results: BulkSendMessageItem[];
}

export interface AppEvent<T = unknown> {
  type: string;
  tenant_id: string;
  instance_id?: string;
  payload: T;
}

export interface CampaignRecipientInput {
  phone: string;
  name?: string;
  variables?: Record<string, string>;
}

export interface ArchiveChatRequest {
  phone: string;
  archive: boolean;
  instance_id?: string;
}

export interface MuteChatRequest {
  phone: string;
  mute: boolean;
  duration_hours?: number;
  instance_id?: string;
}

export interface PinChatRequest {
  phone: string;
  pin: boolean;
  instance_id?: string;
}

export interface MarkChatReadRequest {
  phone: string;
  read: boolean;
  last_message_id?: string;
  instance_id?: string;
}

export interface EditMessageRequest {
  phone: string;
  message_id: string;
  new_message: string;
  instance_id?: string;
}

export interface ForwardMessageRequest {
  phone: string;
  message: string;
  instance_id?: string;
}

export interface StarMessageRequest {
  phone: string;
  message_id: string;
  starred: boolean;
  from_me: boolean;
  instance_id?: string;
}

export interface PairPhoneRequest {
  phone: string;
  instance_id?: string;
}

export interface PairPhoneResponse {
  code: string;
}

export interface Campaign {
  id: string;
  tenant_id: string;
  instance_id: string;
  name: string;
  message: string;
  status:
    | "draft"
    | "scheduled"
    | "running"
    | "completed"
    | "failed"
    | "cancelled";
  scheduled_at?: string;
  last_executed_at?: string;
  total_contacts: number;
  sent_count: number;
  failed_count: number;
  created_at: string;
  updated_at: string;
}
