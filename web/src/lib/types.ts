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
  usage?: Usage;
  plan?: Subscription;
}

export interface Message {
  id: string;
  tenant_id: string;
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

export interface Webhook {
  id: string;
  tenant_id: string;
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

export interface AppEvent<T = unknown> {
  type: string;
  tenant_id: string;
  payload: T;
}
