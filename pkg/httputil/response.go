package httputil

import (
	"encoding/json"
	"errors"
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

// maxBodyBytes is the maximum request body size accepted by Decode.
// Requests larger than this are rejected with 413 before decoding starts.
const maxBodyBytes = 1 << 20 // 1 MiB

// Decode reads JSON body into v, rejecting bodies larger than 1 MiB.
func Decode(r *http.Request, v interface{}) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
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
	case nil:
		return
	}

	switch {
	case errors.Is(err, domain.ErrUnauthorized),
		errors.Is(err, domain.ErrInvalidAPIKey),
		errors.Is(err, domain.ErrInvalidCredentials),
		errors.Is(err, domain.ErrUserSessionExpired):
		Error(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrTenantNotFound),
		errors.Is(err, domain.ErrMessageNotFound),
		errors.Is(err, domain.ErrWebhookNotFound),
		errors.Is(err, domain.ErrWebhookDeliveryNotFound),
		errors.Is(err, domain.ErrQueueJobNotFound),
		errors.Is(err, domain.ErrSessionNotFound),
		errors.Is(err, domain.ErrSessionMetadataNotFound),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrUserSessionNotFound):
		Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrLimitExceeded):
		Error(w, http.StatusPaymentRequired, err.Error())
	case errors.Is(err, domain.ErrConflict),
		errors.Is(err, domain.ErrUserAlreadyInTenant):
		Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrBadRequest),
		errors.Is(err, domain.ErrInvalidPhone),
		errors.Is(err, domain.ErrNoSubscription),
		errors.Is(err, domain.ErrMessageMediaAbsent):
		Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrTenantAccessDenied),
		errors.Is(err, domain.ErrUserRoleForbidden):
		Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrSessionNotConnected),
		errors.Is(err, domain.ErrUserInactive),
		errors.Is(err, domain.ErrTenantInactive):
		Error(w, http.StatusConflict, err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal server error")
	}
}
