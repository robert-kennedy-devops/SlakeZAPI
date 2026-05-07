package handler

import (
	"net/http"
	"strconv"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// ─── Webhook Handler ──────────────────────────────────────────────────────────

type WebhookHandler struct {
	webhookUC *usecase.WebhookUsecase
	billingUC *usecase.BillingUsecase
	log       *logger.Logger
}

func NewWebhookHandler(webhookUC *usecase.WebhookUsecase, billingUC *usecase.BillingUsecase, log *logger.Logger) *WebhookHandler {
	return &WebhookHandler{webhookUC: webhookUC, billingUC: billingUC, log: log}
}

// Register godoc
// POST /webhook
// Body: {"url": "https://myapp.com/hook", "events": ["message.received"]}
func (h *WebhookHandler) Register(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req domain.RegisterWebhookRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.URL == "" {
		httputil.Error(w, http.StatusBadRequest, "url is required")
		return
	}
	if len(req.Events) == 0 {
		req.Events = []string{"message.received", "message.sent", "message.status", "connection.update"}
	}

	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	wh, err := h.webhookUC.Register(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("webhook.register", map[string]interface{}{
		"webhook_id": wh.ID,
		"url":        wh.URL,
		"events":     wh.Events,
	})

	httputil.JSON(w, http.StatusCreated, wh)
}

// List godoc
// GET /webhook
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())

	hooks, err := h.webhookUC.ListByTenant(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()))
	if err != nil {
		httputil.DomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, hooks)
}

func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	if err := h.webhookUC.Delete(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()), r.PathValue("id")); err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("webhook.delete", map[string]interface{}{
		"webhook_id": r.PathValue("id"),
	})

	w.WriteHeader(http.StatusNoContent)
}

func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.webhookUC.ListDeliveries(
		r.Context(),
		tenantID,
		middleware.InstanceFromCtx(r.Context()),
		r.URL.Query().Get("webhook_id"),
		limit,
	)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, items)
}

func (h *WebhookHandler) ReplayDelivery(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	resp, err := h.webhookUC.ReplayDelivery(
		r.Context(),
		tenantID,
		middleware.InstanceFromCtx(r.Context()),
		r.PathValue("id"),
	)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("webhook.replay", map[string]interface{}{
		"delivery_id": resp.DeliveryID,
	})
	httputil.JSON(w, http.StatusAccepted, resp)
}

func (h *WebhookHandler) Usage(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	usage, err := h.billingUC.GetUsage(r.Context(), tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, usage)
}
