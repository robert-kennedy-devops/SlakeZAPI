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
  status: string;
  period_end: string;
  created_at: string;
}

export interface TenantSummary {
  tenant: Tenant;
  session?: SessionStatus;
  instances?: Instance[];
  usage?: Usage;
  plan?: Subscription;
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

export interface Group {
  jid: string;
  name: string;
  topic?: string;
  participant_count: number;
  is_announce: boolean;
  is_locked: boolean;
  created_at: string;
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
