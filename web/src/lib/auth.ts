// Access token lives in module-level memory only — never persisted to
// localStorage so XSS cannot read it via document.localStorage.
// On page reload the dashboard recovers the session via the HttpOnly
// refresh-token cookie by calling POST /app/auth/refresh.
let _accessToken = "";
let _tokenExpiry = "";

const TENANT_KEY = "wsaas.app.tenant";

export function getAuthToken(): string {
  return _accessToken;
}

export function setAuthToken(token: string): void {
  _accessToken = token;
}

export function getAuthTokenExpiry(): string {
  return _tokenExpiry;
}

export function setAuthTokenExpiry(expiresAt: string): void {
  _tokenExpiry = expiresAt;
}

export function clearAuthToken(): void {
  _accessToken = "";
  _tokenExpiry = "";
}

// Tenant ID is not sensitive — it is just a routing hint stored in
// localStorage so the correct workspace is pre-selected after reload.
export function getSelectedTenantID(): string {
  if (typeof window === "undefined") return "";
  return window.localStorage.getItem(TENANT_KEY) ?? "";
}

export function setSelectedTenantID(tenantID: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(TENANT_KEY, tenantID);
}

export function clearSelectedTenantID(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(TENANT_KEY);
}
