package handler

import (
	"net/http"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type UserAuthHandler struct {
	userAuthUC *usecase.UserAuthUsecase
	log        *logger.Logger
}

func NewUserAuthHandler(userAuthUC *usecase.UserAuthUsecase, log *logger.Logger) *UserAuthHandler {
	return &UserAuthHandler{userAuthUC: userAuthUC, log: log}
}

func (h *UserAuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req domain.SignUpRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.userAuthUC.SignUp(r.Context(), req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
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
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.userAuthUC.Login(r.Context(), req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("app.auth.login", map[string]interface{}{
		"user_id":   resp.User.ID,
		"tenant_id": resp.Tenant.ID,
		"email":     req.Email,
	})

	httputil.JSON(w, http.StatusOK, resp)
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
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("app.auth.logout", map[string]interface{}{
		"session_id": sessionID,
	})
	w.WriteHeader(http.StatusNoContent)
}
