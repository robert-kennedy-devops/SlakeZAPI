package handler

import (
	"net/http"
	"strconv"

	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
)

type AuditHandler struct {
	auditUC *usecase.AuditUsecase
}

func NewAuditHandler(auditUC *usecase.AuditUsecase) *AuditHandler {
	return &AuditHandler{auditUC: auditUC}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	items, err := h.auditUC.List(
		r.Context(),
		tenantID,
		middleware.InstanceFromCtx(r.Context()),
		r.URL.Query().Get("action_prefix"),
		limit,
	)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, items)
}
