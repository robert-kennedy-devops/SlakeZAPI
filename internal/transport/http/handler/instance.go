package handler

import (
	"net/http"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type InstanceHandler struct {
	instanceUC *usecase.InstanceUsecase
	log        *logger.Logger
}

func NewInstanceHandler(instanceUC *usecase.InstanceUsecase, log *logger.Logger) *InstanceHandler {
	return &InstanceHandler{instanceUC: instanceUC, log: log}
}

func (h *InstanceHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	instances, err := h.instanceUC.ListByTenant(r.Context(), tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, instances)
}

func (h *InstanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req domain.CreateInstanceRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	instance, err := h.instanceUC.Create(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("instance.create", map[string]interface{}{
		"instance_id": instance.ID,
		"name":        instance.Name,
	})
	httputil.JSON(w, http.StatusCreated, instance)
}
