package handler

import (
	"net/http"
	"strconv"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
)

type OperationsHandler struct {
	opsUC *usecase.OperationsUsecase
}

func NewOperationsHandler(opsUC *usecase.OperationsUsecase) *OperationsHandler {
	return &OperationsHandler{opsUC: opsUC}
}

func (h *OperationsHandler) Queue(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, http.StatusOK, h.opsUC.QueueSnapshot())
}

func (h *OperationsHandler) DeadLetters(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	httputil.JSON(w, http.StatusOK, h.opsUC.DeadLetters(limit))
}

func (h *OperationsHandler) RequeueDeadLetter(w http.ResponseWriter, r *http.Request) {
	if err := h.opsUC.RequeueDeadLetter(r.PathValue("id")); err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusAccepted, domain.QueueRequeueResponse{
		JobID:  r.PathValue("id"),
		Status: "requeued",
	})
}
