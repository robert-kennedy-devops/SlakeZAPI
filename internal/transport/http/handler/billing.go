package handler

import (
	"io"
	"net/http"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type BillingHandler struct {
	billingUC *usecase.BillingUsecase
	log       *logger.Logger
}

func NewBillingHandler(billingUC *usecase.BillingUsecase, log *logger.Logger) *BillingHandler {
	return &BillingHandler{billingUC: billingUC, log: log}
}

func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	sub, err := h.billingUC.GetSubscription(r.Context(), tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, sub)
}

func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req domain.CreateCheckoutRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.billingUC.CreateCheckout(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("billing.checkout.create", map[string]interface{}{
		"tenant_id": tenantID,
		"plan":      req.Plan,
	})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *BillingHandler) OpenPortal(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.billingUC.OpenPortal(r.Context(), tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("billing.portal.open", map[string]interface{}{
		"tenant_id": tenantID,
	})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *BillingHandler) ChangePlan(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req domain.ChangeSubscriptionPlanRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.billingUC.ChangePlan(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("billing.plan.change", map[string]interface{}{
		"tenant_id": tenantID,
		"plan":      req.Plan,
	})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.billingUC.CancelSubscription(r.Context(), tenantID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("billing.subscription.cancel", map[string]interface{}{
		"tenant_id": tenantID,
	})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *BillingHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}

	if err := h.billingUC.HandleWebhook(r.Context(), payload, r.Header.Get("Stripe-Signature")); err != nil {
		httputil.DomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
