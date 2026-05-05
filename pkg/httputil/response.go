package httputil

import (
	"encoding/json"
	"net/http"

	"github.com/whatsapp-saas/api/internal/domain"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// JSON writes v as JSON with the given status code.
func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Decode reads JSON body into v.
func Decode(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// Error writes a structured error response.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, ErrorResponse{Error: msg, Code: status})
}

// DomainError maps domain errors to HTTP status codes.
func DomainError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrUnauthorized, domain.ErrInvalidAPIKey:
		Error(w, http.StatusUnauthorized, err.Error())
	case domain.ErrTenantNotFound, domain.ErrMessageNotFound,
		domain.ErrWebhookNotFound, domain.ErrSessionNotFound, domain.ErrSessionMetadataNotFound:
		Error(w, http.StatusNotFound, err.Error())
	case domain.ErrLimitExceeded:
		Error(w, http.StatusPaymentRequired, err.Error())
	case domain.ErrConflict:
		Error(w, http.StatusConflict, err.Error())
	case domain.ErrBadRequest, domain.ErrInvalidPhone, domain.ErrNoSubscription, domain.ErrMessageMediaAbsent:
		Error(w, http.StatusBadRequest, err.Error())
	case domain.ErrSessionNotConnected:
		Error(w, http.StatusConflict, err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal server error")
	}
}
