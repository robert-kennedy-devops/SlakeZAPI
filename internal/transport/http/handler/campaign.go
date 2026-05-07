package handler

import (
	"net/http"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type CampaignHandler struct {
	campaignUC *usecase.CampaignUsecase
	log        *logger.Logger
}

func NewCampaignHandler(campaignUC *usecase.CampaignUsecase, log *logger.Logger) *CampaignHandler {
	return &CampaignHandler{campaignUC: campaignUC, log: log}
}

func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	campaigns, err := h.campaignUC.List(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()))
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, campaigns)
}

func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.CreateCampaignRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.InstanceID == "" {
		req.InstanceID = middleware.InstanceFromCtx(r.Context())
	}
	campaign, err := h.campaignUC.Create(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("campaign.create", map[string]interface{}{
		"campaign_id": campaign.ID,
		"instance_id": campaign.InstanceID,
		"status":      campaign.Status,
	})
	httputil.JSON(w, http.StatusCreated, campaign)
}

func (h *CampaignHandler) Run(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	campaignID := r.PathValue("id")
	if campaignID == "" {
		httputil.Error(w, http.StatusBadRequest, "missing campaign id")
		return
	}
	if err := h.campaignUC.Run(r.Context(), tenantID, campaignID); err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("campaign.run", map[string]interface{}{
		"campaign_id": campaignID,
	})
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "started"})
}
