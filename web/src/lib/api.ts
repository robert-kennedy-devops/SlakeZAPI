import type {
  APIKey,
  AppEvent,
  CreateAPIKeyResponse,
  CurrentUserResponse,
  Message,
  SendMessageResponse,
  SessionStatus,
  TenantSummary,
  Usage,
  UserSessionResponse,
  Webhook,
} from "@/lib/types";

type RequestOptions = {
  token?: string;
  tenantID?: string;
  method?: string;
  body?: unknown;
};

export function getAPIBaseURL() {
  if (typeof window !== "undefined") {
    return process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  }
  return process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
}

function toHeaders(token?: string, tenantID?: string, body?: unknown) {
  const headers = new Headers();
  if (body !== undefined) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (tenantID) headers.set("X-Tenant-ID", tenantID);
  return headers;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await fetch(`${getAPIBaseURL()}${path}`, {
    method: options.method ?? "GET",
    headers: toHeaders(options.token, options.tenantID, options.body),
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `request failed with status ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export const api = {
  signUp: (body: { name: string; email: string; password: string; tenant_name: string; plan: string }) =>
    request<UserSessionResponse>("/app/auth/signup", { method: "POST", body }),
  login: (body: { email: string; password: string }) =>
    request<UserSessionResponse>("/app/auth/login", { method: "POST", body }),
  logout: (token: string) => request<void>("/app/auth/logout", { method: "POST", token }),
  me: (token: string, tenantID?: string) => request<CurrentUserResponse>("/app/auth/me", { token, tenantID }),
  tenantSummary: (token: string, tenantID: string) =>
    request<TenantSummary>("/app/tenant/summary", { token, tenantID }),
  whatsappStatus: (token: string, tenantID: string) =>
    request<SessionStatus>("/app/whatsapp/status", { token, tenantID }),
  whatsappConnect: (token: string, tenantID: string) =>
    request<SessionStatus & { qr_png_url?: string; qr_page_url?: string }>("/app/whatsapp/connect", { method: "POST", token, tenantID }),
  whatsappDisconnect: (token: string, tenantID: string) =>
    request<{ status: string }>("/app/whatsapp/disconnect", { method: "POST", token, tenantID }),
  whatsappLogout: (token: string, tenantID: string) =>
    request<{ status: string }>("/app/whatsapp/logout", { method: "POST", token, tenantID }),
  messages: (token: string, tenantID: string) =>
    request<Message[]>("/app/messages", { token, tenantID }),
  sendMessage: (token: string, tenantID: string, body: { phone: string; message: string }) =>
    request<SendMessageResponse>("/app/messages/send", { method: "POST", token, tenantID, body }),
  sendMedia: (
    token: string,
    tenantID: string,
    body: { phone: string; type: string; url: string; caption?: string; file_name?: string; mime_type?: string },
  ) => request<SendMessageResponse>("/app/messages/send-media", { method: "POST", token, tenantID, body }),
  webhooks: (token: string, tenantID: string) =>
    request<Webhook[]>("/app/webhooks", { token, tenantID }),
  createWebhook: (token: string, tenantID: string, body: { url: string; events: string[] }) =>
    request<Webhook>("/app/webhooks", { method: "POST", token, tenantID, body }),
  deleteWebhook: (token: string, tenantID: string, id: string) =>
    request<void>(`/app/webhooks/${id}`, { method: "DELETE", token, tenantID }),
  usage: (token: string, tenantID: string) =>
    request<Usage>("/app/usage", { token, tenantID }),
  apiKeys: (token: string, tenantID: string) =>
    request<APIKey[]>("/app/apikeys", { token, tenantID }),
  createAPIKey: (token: string, tenantID: string, body: { label: string }) =>
    request<CreateAPIKeyResponse>("/app/apikeys", { method: "POST", token, tenantID, body }),
  deleteAPIKey: (token: string, tenantID: string, id: string) =>
    request<void>(`/app/apikeys/${id}`, { method: "DELETE", token, tenantID }),
};

export function makeWSURL(token: string, tenantID: string) {
  const apiBase = getAPIBaseURL();
  const url = new URL("/app/ws", apiBase);
  url.searchParams.set("access_token", token);
  url.searchParams.set("tenant_id", tenantID);
  if (url.protocol === "http:") url.protocol = "ws:";
  if (url.protocol === "https:") url.protocol = "wss:";
  return url.toString();
}

export function makeQRImageURL(token: string, tenantID: string) {
  const url = new URL("/app/whatsapp/qr.png", getAPIBaseURL());
  url.searchParams.set("access_token", token);
  url.searchParams.set("tenant_id", tenantID);
  return url.toString();
}

export function makeMediaDownloadURL(token: string, tenantID: string, messageID: string) {
  const url = new URL(`/app/messages/${messageID}/media`, getAPIBaseURL());
  url.searchParams.set("access_token", token);
  url.searchParams.set("tenant_id", tenantID);
  return url.toString();
}

export type { AppEvent };
