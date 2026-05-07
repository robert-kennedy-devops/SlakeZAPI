package domain

import "errors"

var (
	// Auth
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidAPIKey       = errors.New("invalid or expired API key")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserSessionNotFound = errors.New("user session not found")
	ErrUserSessionExpired  = errors.New("user session expired")
	ErrUserRoleForbidden   = errors.New("user role forbidden")

	// Tenant
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrTenantInactive     = errors.New("tenant is inactive")
	ErrTenantAccessDenied = errors.New("tenant access denied")

	// Users
	ErrUserNotFound = errors.New("user not found")
	ErrUserInactive = errors.New("user is inactive")

	// Billing
	ErrLimitExceeded  = errors.New("monthly message limit exceeded")
	ErrNoSubscription = errors.New("no active subscription found")

	// WhatsApp
	ErrSessionNotFound     = errors.New("whatsapp session not found")
	ErrSessionNotConnected = errors.New("whatsapp session not connected")
	ErrQRTimeout           = errors.New("QR code scan timeout")

	// Messages
	ErrMessageNotFound    = errors.New("message not found")
	ErrMessageMediaAbsent = errors.New("message has no downloadable media")
	ErrInvalidPhone       = errors.New("invalid phone number")

	// Webhook
	ErrWebhookNotFound = errors.New("webhook not found")

	// Session metadata
	ErrSessionMetadataNotFound = errors.New("whatsapp session metadata not found")

	// Generic
	ErrConflict   = errors.New("resource conflict")
	ErrNotFound   = errors.New("resource not found")
	ErrBadRequest = errors.New("bad request")
	ErrInternal   = errors.New("internal server error")
)
