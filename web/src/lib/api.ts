import type {
  APIKey,
  AuditLog,
  AppEvent,
  CreateAPIKeyResponse,
  Conversation,
  CurrentUserResponse,
  Campaign,
  Group,
  Instance,
  Message,
  BulkSendMessageResponse,
  QueueJobView,
  QueueRequeueResponse,
  QueueSnapshot,
  ResolvedContact,
  SendMessageResponse,
  SessionStatus,
  TenantMember,
  TenantSummary,
  Usage,
  UserSessionResponse,
  Webhook,
  WebhookDelivery,
} from "@/lib/types";
import {
  clearAuthToken,
  getAuthToken,
  setAuthToken,
  setAuthTokenExpiry,
} from "@/lib/auth";

type RequestOptions = {
  token?: string;
  tenantID?: string;
  instanceID?: string;
  method?: string;
  body?: unknown;
  skipRefresh?: boolean;
};

type APIErrorPayload = {
  error?: string;
  message?: string;
};

export function getAPIBaseURL() {
  if (typeof window !== "undefined") {
    return process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  }
  return process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
}

function toHeaders(token?: string, tenantID?: string, instanceID?: string, body?: unknown) {
  const headers = new Headers();
  if (body !== undefined) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (tenantID) headers.set("X-Tenant-ID", tenantID);
  if (instanceID) headers.set("X-Instance-ID", instanceID);
  return headers;
}

async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const response = await fetch(`${getAPIBaseURL()}${path}`, {
    method: options.method ?? "GET",
    headers: toHeaders(options.token, options.tenantID, options.instanceID, options.body),
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    credentials: "include",
  });

  if (response.status === 401 && !options.skipRefresh) {
    const token = options.token ?? getAuthToken();
    if (token) {
      try {
        await refreshSession(options.tenantID);
        return request<T>(path, {
          ...options,
          token: getAuthToken(),
          skipRefresh: true,
        });
      } catch {
        clearAuthToken();
      }
    }
  }

  if (!response.ok) {
    const message = await readErrorMessage(response);
    throw new Error(message || `request failed with status ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

async function readErrorMessage(response: Response) {
  const contentType = response.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json")) {
    const payload = (await response.json().catch(() => null)) as
      | APIErrorPayload
      | null;
    if (payload?.error) return payload.error;
    if (payload?.message) return payload.message;
  }

  return response.text();
}

let refreshPromise: Promise<UserSessionResponse> | null = null;

async function refreshSession(tenantID?: string) {
  if (!refreshPromise) {
    refreshPromise = request<UserSessionResponse>("/app/auth/refresh", {
      method: "POST",
      tenantID,
      skipRefresh: true,
    })
      .then((session) => {
        setAuthToken(session.token);
        setAuthTokenExpiry(session.expires_at);
        return session;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }

  return refreshPromise;
}

export const api = {
  signUp: (body: {
    name: string;
    email: string;
    password: string;
    tenant_name: string;
    plan: string;
  }) =>
    request<UserSessionResponse>("/app/auth/signup", { method: "POST", body }),
  login: (body: { email: string; password: string }) =>
    request<UserSessionResponse>("/app/auth/login", { method: "POST", body }),
  refresh: (tenantID?: string) => refreshSession(tenantID),
  logout: (token: string) =>
    request<void>("/app/auth/logout", { method: "POST", token }),
  me: (token: string, tenantID?: string) =>
    request<CurrentUserResponse>("/app/auth/me", { token, tenantID }),
  tenantSummary: (token: string, tenantID: string) =>
    request<TenantSummary>("/app/tenant/summary", { token, tenantID }),
  instances: (token: string, tenantID: string) =>
    request<Instance[]>("/app/instances", { token, tenantID }),
  createInstance: (
    token: string,
    tenantID: string,
    body: { name: string },
  ) =>
    request<Instance>("/app/instances", {
      method: "POST",
      token,
      tenantID,
      body,
    }),
  whatsappStatus: (token: string, tenantID: string, instanceID?: string) =>
    request<SessionStatus>("/app/whatsapp/status", { token, tenantID, instanceID }),
  whatsappConnect: (token: string, tenantID: string, instanceID?: string) =>
    request<SessionStatus & { qr_png_url?: string; qr_page_url?: string }>(
      "/app/whatsapp/connect",
      { method: "POST", token, tenantID, instanceID },
    ),
  whatsappDisconnect: (token: string, tenantID: string, instanceID?: string) =>
    request<{ status: string }>("/app/whatsapp/disconnect", {
      method: "POST",
      token,
      tenantID,
      instanceID,
    }),
  whatsappLogout: (token: string, tenantID: string, instanceID?: string) =>
    request<{ status: string }>("/app/whatsapp/logout", {
      method: "POST",
      token,
      tenantID,
      instanceID,
    }),
  messages: (token: string, tenantID: string, instanceID?: string) =>
    request<Message[]>("/app/messages", { token, tenantID, instanceID }),
  conversations: (token: string, tenantID: string, instanceID?: string) =>
    request<Conversation[]>("/app/conversations", { token, tenantID, instanceID }),
  updateConversation: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    phone: string,
    body: { state?: "open" | "pending" | "resolved"; note?: string },
  ) =>
    request<Conversation>(`/app/conversations/${encodeURIComponent(phone)}`, {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  sendMessage: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: { phone: string; message: string },
  ) =>
    request<SendMessageResponse>("/app/messages/send", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  sendMedia: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: {
      phone: string;
      type: string;
      url: string;
      caption?: string;
      file_name?: string;
      mime_type?: string;
    },
  ) =>
    request<SendMessageResponse>("/app/messages/send-media", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  sendInteractiveMessage: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: {
      phone: string;
      type: string;
      header?: string;
      body: string;
      footer?: string;
      button_text?: string;
      buttons?: { id: string; title: string }[];
      sections?: { title: string; rows: { id: string; title: string; description?: string }[] }[];
      options?: string[];
      max_select?: number;
    },
  ) =>
    request<SendMessageResponse>("/app/messages/send-interactive", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  sendGroupMessage: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: { group_jid: string; message: string },
  ) =>
    request<SendMessageResponse>("/app/messages/send-group", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  postStatus: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: {
      type: string;
      message?: string;
      url?: string;
      caption?: string;
      file_name?: string;
      mime_type?: string;
    },
  ) =>
    request<SendMessageResponse>("/app/status/post", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  groups: (token: string, tenantID: string, instanceID?: string) =>
    request<Group[]>("/app/groups", { token, tenantID, instanceID }),
  resolveContacts: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: { phones: string[] },
  ) =>
    request<ResolvedContact[]>("/app/contacts/resolve", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  sendBulkMessage: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: { phones: string[]; message: string },
  ) =>
    request<BulkSendMessageResponse>("/app/messages/send-bulk", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  campaigns: (token: string, tenantID: string, instanceID?: string) =>
    request<Campaign[]>("/app/campaigns", { token, tenantID, instanceID }),
  createCampaign: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: {
      name: string;
      message: string;
      scheduled_at?: string;
      recipients: { phone: string; name?: string; variables?: Record<string, string> }[];
    },
  ) =>
    request<Campaign>("/app/campaigns", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  runCampaign: (token: string, tenantID: string, instanceID: string | undefined, id: string) =>
    request<{ status: string }>(`/app/campaigns/${id}/run`, {
      method: "POST",
      token,
      tenantID,
      instanceID,
    }),
  webhooks: (token: string, tenantID: string, instanceID?: string) =>
    request<Webhook[]>("/app/webhooks", { token, tenantID, instanceID }),
  webhookDeliveries: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    webhookID?: string,
  ) =>
    request<WebhookDelivery[]>(
      `/app/webhooks/deliveries${webhookID ? `?webhook_id=${encodeURIComponent(webhookID)}` : ""}`,
      { token, tenantID, instanceID },
    ),
  createWebhook: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    body: { url: string; events: string[] },
  ) =>
    request<Webhook>("/app/webhooks", {
      method: "POST",
      token,
      tenantID,
      instanceID,
      body,
    }),
  deleteWebhook: (token: string, tenantID: string, instanceID: string | undefined, id: string) =>
    request<void>(`/app/webhooks/${id}`, { method: "DELETE", token, tenantID, instanceID }),
  replayWebhookDelivery: (
    token: string,
    tenantID: string,
    instanceID: string | undefined,
    id: string,
  ) =>
    request<{ delivery_id: string; status: string }>(`/app/webhooks/deliveries/${id}/replay`, {
      method: "POST",
      token,
      tenantID,
      instanceID,
    }),
  usage: (token: string, tenantID: string) =>
    request<Usage>("/app/usage", { token, tenantID }),
  queue: (token: string, tenantID: string) =>
    request<QueueSnapshot>("/app/queue", { token, tenantID }),
  queueDeadLetters: (token: string, tenantID: string) =>
    request<QueueJobView[]>("/app/queue/dead-letters", { token, tenantID }),
  requeueDeadLetter: (token: string, tenantID: string, id: string) =>
    request<QueueRequeueResponse>(`/app/queue/dead-letters/${encodeURIComponent(id)}/requeue`, {
      method: "POST",
      token,
      tenantID,
    }),
  members: (token: string, tenantID: string) =>
    request<TenantMember[]>("/app/members", { token, tenantID }),
  auditLogs: (token: string, tenantID: string, instanceID?: string) =>
    request<AuditLog[]>("/app/audit", { token, tenantID, instanceID }),
  addMember: (
    token: string,
    tenantID: string,
    body: { email: string; role: string },
  ) =>
    request<TenantMember>("/app/members", {
      method: "POST",
      token,
      tenantID,
      body,
    }),
  updateMemberRole: (
    token: string,
    tenantID: string,
    id: string,
    body: { role: string },
  ) =>
    request<TenantMember>(`/app/members/${id}/role`, {
      method: "POST",
      token,
      tenantID,
      body,
    }),
  apiKeys: (token: string, tenantID: string) =>
    request<APIKey[]>("/app/apikeys", { token, tenantID }),
  createAPIKey: (token: string, tenantID: string, body: { label: string }) =>
    request<CreateAPIKeyResponse>("/app/apikeys", {
      method: "POST",
      token,
      tenantID,
      body,
    }),
  deleteAPIKey: (token: string, tenantID: string, id: string) =>
    request<void>(`/app/apikeys/${id}`, { method: "DELETE", token, tenantID }),
};

// Token is intentionally omitted from these URLs — the browser sends the
// HttpOnly access-token cookie automatically on WebSocket upgrades and
// resource loads (<img>, download links), so no credential ends up in
// server logs, Referer headers, or browser history.
export function makeWSURL(tenantID: string, instanceID?: string) {
  const apiBase = getAPIBaseURL();
  const url = new URL("/app/ws", apiBase);
  url.searchParams.set("tenant_id", tenantID);
  if (instanceID) url.searchParams.set("instance_id", instanceID);
  if (url.protocol === "http:") url.protocol = "ws:";
  if (url.protocol === "https:") url.protocol = "wss:";
  return url.toString();
}

export function makeQRImageURL(tenantID: string, instanceID?: string) {
  const url = new URL("/app/whatsapp/qr.png", getAPIBaseURL());
  url.searchParams.set("tenant_id", tenantID);
  if (instanceID) url.searchParams.set("instance_id", instanceID);
  return url.toString();
}

export function makeMediaDownloadURL(
  tenantID: string,
  instanceID: string | undefined,
  messageID: string,
) {
  const url = new URL(`/app/messages/${messageID}/media`, getAPIBaseURL());
  url.searchParams.set("tenant_id", tenantID);
  if (instanceID) url.searchParams.set("instance_id", instanceID);
  return url.toString();
}

export function makeAPIDocURL(path: "openapi.yaml" | "postman_collection.json") {
  return new URL(`/docs/${path}`, getAPIBaseURL()).toString();
}

export type { AppEvent };
