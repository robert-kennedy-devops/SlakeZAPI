const TOKEN_KEY = "wsaas.app.token";
const TOKEN_EXPIRY_KEY = "wsaas.app.token_expiry";
const TENANT_KEY = "wsaas.app.tenant";

export function getAuthToken() {
  if (typeof window === "undefined") return "";
  return window.localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setAuthToken(token: string) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function getAuthTokenExpiry() {
  if (typeof window === "undefined") return "";
  return window.localStorage.getItem(TOKEN_EXPIRY_KEY) ?? "";
}

export function setAuthTokenExpiry(expiresAt: string) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TOKEN_EXPIRY_KEY, expiresAt);
}

export function clearAuthToken() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TOKEN_KEY);
  window.localStorage.removeItem(TOKEN_EXPIRY_KEY);
}

export function getSelectedTenantID() {
  if (typeof window === "undefined") return "";
  return window.localStorage.getItem(TENANT_KEY) ?? "";
}

export function setSelectedTenantID(tenantID: string) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TENANT_KEY, tenantID);
}

export function clearSelectedTenantID() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TENANT_KEY);
}
