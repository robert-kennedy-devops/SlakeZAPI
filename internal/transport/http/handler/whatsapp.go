package handler

import (
	"net/http"

	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// WhatsAppHandler exposes WhatsApp session endpoints.
type WhatsAppHandler struct {
	waUC *usecase.WhatsAppUsecase
	log  *logger.Logger
}

func NewWhatsAppHandler(waUC *usecase.WhatsAppUsecase, log *logger.Logger) *WhatsAppHandler {
	return &WhatsAppHandler{waUC: waUC, log: log}
}

// Connect godoc
// POST /whatsapp/connect
// Header: Authorization: Bearer <api_key>
// Response: {"qr_code": "...", "status": "connecting"}
func (h *WhatsAppHandler) Connect(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.waUC.Connect(r.Context(), tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("whatsapp.connect", map[string]interface{}{
		"status": resp.Status,
		"phone":  resp.Phone,
	})

	httputil.JSON(w, http.StatusOK, resp)
}

// Status godoc
// GET /whatsapp/status
func (h *WhatsAppHandler) Status(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	session, err := h.waUC.GetSession(r.Context(), tenantID)
	if err == nil {
		httputil.JSON(w, http.StatusOK, session)
		return
	}

	status := h.waUC.GetStatus(r.Context(), tenantID)
	httputil.JSON(w, http.StatusOK, map[string]string{"status": string(status)})
}

func (h *WhatsAppHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	if err := h.waUC.Disconnect(r.Context(), tenantID); err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("whatsapp.disconnect")

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

func (h *WhatsAppHandler) Logout(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	if err := h.waUC.Logout(r.Context(), tenantID); err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("whatsapp.logout")

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}
