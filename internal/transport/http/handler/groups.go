package handler

import (
	"net/http"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
)

type GroupHandler struct {
	waUC  *usecase.WhatsAppUsecase
	msgUC *usecase.MessageUsecase
}

func NewGroupHandler(waUC *usecase.WhatsAppUsecase, msgUC *usecase.MessageUsecase) *GroupHandler {
	return &GroupHandler{waUC: waUC, msgUC: msgUC}
}

func (h *GroupHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	groups, err := h.msgUC.ListGroups(r.Context(), tenantID, instanceID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, groups)
}

func (h *GroupHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	groupJID := r.PathValue("jid")
	if groupJID == "" {
		httputil.Error(w, http.StatusBadRequest, "group jid is required")
		return
	}
	group, err := h.waUC.GetGroupInfo(r.Context(), tenantID, instanceID, groupJID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, group)
}

func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.CreateGroupRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	group, err := h.waUC.CreateGroup(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, group)
}

func (h *GroupHandler) UpdateParticipants(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	groupJID := r.PathValue("jid")
	if groupJID == "" {
		httputil.Error(w, http.StatusBadRequest, "group jid is required")
		return
	}
	var req domain.UpdateGroupParticipantsRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	req.GroupJID = groupJID
	if err := h.waUC.UpdateGroupParticipants(r.Context(), tenantID, req); err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *GroupHandler) UpdateInfo(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	groupJID := r.PathValue("jid")
	if groupJID == "" {
		httputil.Error(w, http.StatusBadRequest, "group jid is required")
		return
	}
	var req domain.UpdateGroupInfoRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	req.GroupJID = groupJID
	if err := h.waUC.UpdateGroupInfo(r.Context(), tenantID, req); err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *GroupHandler) InviteLink(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	groupJID := r.PathValue("jid")
	if groupJID == "" {
		httputil.Error(w, http.StatusBadRequest, "group jid is required")
		return
	}
	link, err := h.waUC.GetGroupInviteLink(r.Context(), tenantID, instanceID, groupJID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, link)
}

func (h *GroupHandler) Leave(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	groupJID := r.PathValue("jid")
	if groupJID == "" {
		httputil.Error(w, http.StatusBadRequest, "group jid is required")
		return
	}
	if err := h.waUC.LeaveGroup(r.Context(), tenantID, instanceID, groupJID); err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "left"})
}
