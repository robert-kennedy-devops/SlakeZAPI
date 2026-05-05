package handler

import (
	"net/http"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// AuthHandler exposes auth-related endpoints.
type AuthHandler struct {
	authUC *usecase.AuthUsecase
	log    *logger.Logger
}

func NewAuthHandler(authUC *usecase.AuthUsecase, log *logger.Logger) *AuthHandler {
	return &AuthHandler{authUC: authUC, log: log}
}

// CreateAPIKey godoc
// POST /auth/apikey
// Body: {"label": "my key"}
// Header: Authorization: Bearer <existing_key>  (or admin token in prod)
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req struct {
		Label string `json:"label"`
	}
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Label == "" {
		req.Label = "default"
	}

	resp, err := h.authUC.CreateAPIKey(r.Context(), tenantID, req.Label)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("auth.apikey.create", map[string]interface{}{
		"key_id": resp.KeyID,
		"label":  req.Label,
	})

	httputil.JSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Plan  string `json:"plan"`
	}
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authUC.BootstrapTenant(r.Context(), domain.BootstrapTenantRequest{
		Name:  req.Name,
		Email: req.Email,
		Plan:  domain.PlanName(req.Plan),
	})
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("auth.bootstrap", map[string]interface{}{
		"tenant_id": resp.TenantID,
		"plan":      resp.Plan,
		"email":     req.Email,
	})

	httputil.JSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.authUC.ListAPIKeys(r.Context(), tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	if err := h.authUC.RevokeAPIKey(r.Context(), tenantID, r.PathValue("id")); err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("auth.apikey.revoke", map[string]interface{}{
		"key_id": r.PathValue("id"),
	})

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.authUC.GetTenantSummary(r.Context(), tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}
