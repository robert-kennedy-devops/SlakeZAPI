package handler

import (
	"net/http"

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
