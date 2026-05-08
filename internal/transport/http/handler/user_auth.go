package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/observability"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type UserAuthHandler struct {
	userAuthUC             *usecase.UserAuthUsecase
	log                    *logger.Logger
	metrics                *observability.Metrics
	sessionCookieName      string
	accessTokenCookieName  string
	sessionCookieSecure    bool
	sessionCookieDomain    string
	sessionCookieSameSite  http.SameSite
}

func NewUserAuthHandler(
	userAuthUC *usecase.UserAuthUsecase,
	metrics *observability.Metrics,
	sessionCookieName string,
	accessTokenCookieName string,
	sessionCookieSecure bool,
	sessionCookieDomain string,
	sessionCookieSameSite string,
	log *logger.Logger,
) *UserAuthHandler {
	return &UserAuthHandler{
		userAuthUC:            userAuthUC,
		metrics:               metrics,
		log:                   log,
		sessionCookieName:     sessionCookieName,
		accessTokenCookieName: accessTokenCookieName,
		sessionCookieSecure:   sessionCookieSecure,
		sessionCookieDomain:   sessionCookieDomain,
		sessionCookieSameSite: parseSameSite(sessionCookieSameSite),
	}
}

func (h *UserAuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req domain.SignUpRequest
	if err := httputil.Decode(r, &req); err != nil {
		h.trackAuth("signup", "bad_request")
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, refreshToken, err := h.userAuthUC.SignUp(r.Context(), req)
	if err != nil {
		h.trackAuth("signup", "error")
		httputil.DomainError(w, err)
		return
	}
	h.trackAuth("signup", "success")
	h.setRefreshCookie(w, refreshToken, resp.RefreshExpiresAt)
	h.setAccessTokenCookie(w, resp.Token, resp.ExpiresAt)
	h.log.WithContext(r.Context()).Audit("app.auth.signup", map[string]interface{}{
		"user_id":   resp.User.ID,
		"tenant_id": resp.Tenant.ID,
		"email":     req.Email,
	})

	httputil.JSON(w, http.StatusCreated, resp)
}

func (h *UserAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := httputil.Decode(r, &req); err != nil {
		h.trackAuth("login", "bad_request")
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, refreshToken, err := h.userAuthUC.Login(r.Context(), req)
	if err != nil {
		h.trackAuth("login", "error")
		httputil.DomainError(w, err)
		return
	}
	h.trackAuth("login", "success")
	h.setRefreshCookie(w, refreshToken, resp.RefreshExpiresAt)
	h.setAccessTokenCookie(w, resp.Token, resp.ExpiresAt)
	h.log.WithContext(r.Context()).Audit("app.auth.login", map[string]interface{}{
		"user_id":   resp.User.ID,
		"tenant_id": resp.Tenant.ID,
		"email":     req.Email,
	})

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *UserAuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing user context")
		return
	}
	var req domain.UpdateUserProfileRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.userAuthUC.UpdateUserProfile(r.Context(), userID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("app.auth.update_profile", map[string]interface{}{
		"user_id": userID,
	})
	httputil.JSON(w, http.StatusOK, user)
}

func (h *UserAuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	tenantID := middleware.TenantFromCtx(r.Context())
	if userID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing user context")
		return
	}

	resp, err := h.userAuthUC.GetCurrentUser(r.Context(), userID, tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *UserAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID := middleware.UserSessionIDFromCtx(r.Context())
	if sessionID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing session context")
		return
	}

	if err := h.userAuthUC.Logout(r.Context(), sessionID); err != nil {
		h.trackAuth("logout", "error")
		httputil.DomainError(w, err)
		return
	}
	h.trackAuth("logout", "success")
	h.log.WithContext(r.Context()).Audit("app.auth.logout", map[string]interface{}{
		"session_id": sessionID,
	})
	h.clearRefreshCookie(w)
	h.clearAccessTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *UserAuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.sessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		h.trackAuth("refresh", "missing_cookie")
		httputil.Error(w, http.StatusUnauthorized, "missing refresh cookie")
		return
	}

	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	}

	resp, nextRefreshToken, err := h.userAuthUC.RefreshSession(r.Context(), cookie.Value, tenantID)
	if err != nil {
		h.trackAuth("refresh", "error")
		h.clearRefreshCookie(w)
		httputil.DomainError(w, err)
		return
	}
	h.trackAuth("refresh", "success")
	h.setRefreshCookie(w, nextRefreshToken, resp.RefreshExpiresAt)
	h.setAccessTokenCookie(w, resp.Token, resp.ExpiresAt)
	h.log.WithContext(r.Context()).Audit("app.auth.refresh", map[string]interface{}{
		"user_id":   resp.User.ID,
		"tenant_id": resp.Tenant.ID,
	})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *UserAuthHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	tenantID := middleware.TenantFromCtx(r.Context())
	if userID == "" || tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing user or tenant context")
		return
	}

	resp, err := h.userAuthUC.ListTenantMembers(r.Context(), userID, tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *UserAuthHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	tenantID := middleware.TenantFromCtx(r.Context())
	if userID == "" || tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing user or tenant context")
		return
	}

	var req domain.AddTenantMemberRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.userAuthUC.AddTenantMember(r.Context(), userID, tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("app.members.add", map[string]interface{}{
		"tenant_id": tenantID,
		"email":     req.Email,
		"role":      req.Role,
	})
	httputil.JSON(w, http.StatusCreated, resp)
}

func (h *UserAuthHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	tenantID := middleware.TenantFromCtx(r.Context())
	if userID == "" || tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing user or tenant context")
		return
	}

	var req domain.UpdateTenantMemberRoleRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.userAuthUC.UpdateTenantMemberRole(r.Context(), userID, tenantID, r.PathValue("id"), req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("app.members.role_update", map[string]interface{}{
		"tenant_id": tenantID,
		"member_id": r.PathValue("id"),
		"role":      req.Role,
	})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *UserAuthHandler) setAccessTokenCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	if h.accessTokenCookieName == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.accessTokenCookieName,
		Value:    token,
		Path:     "/",
		Domain:   h.sessionCookieDomain,
		HttpOnly: true,
		Secure:   h.sessionCookieSecure,
		SameSite: h.sessionCookieSameSite,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func (h *UserAuthHandler) clearAccessTokenCookie(w http.ResponseWriter) {
	if h.accessTokenCookieName == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     h.accessTokenCookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.sessionCookieDomain,
		HttpOnly: true,
		Secure:   h.sessionCookieSecure,
		SameSite: h.sessionCookieSameSite,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

func (h *UserAuthHandler) setRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.sessionCookieName,
		Value:    token,
		Path:     "/",
		Domain:   h.sessionCookieDomain,
		HttpOnly: true,
		Secure:   h.sessionCookieSecure,
		SameSite: h.sessionCookieSameSite,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func (h *UserAuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.sessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.sessionCookieDomain,
		HttpOnly: true,
		Secure:   h.sessionCookieSecure,
		SameSite: h.sessionCookieSameSite,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

func (h *UserAuthHandler) trackAuth(action, outcome string) {
	if h.metrics == nil {
		return
	}
	h.metrics.TrackAuth(action, outcome)
}

func parseSameSite(raw string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
