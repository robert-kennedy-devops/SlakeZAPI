package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type userCtxKey struct{}
type sessionCtxKey struct{}
type userRoleCtxKey struct{}

func UserAuth(userAuthUC *usecase.UserAuthUsecase, log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := extractBearer(r)
			if raw == "" {
				httputil.Error(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			session, err := userAuthUC.ValidateSessionToken(r.Context(), raw)
			if err != nil {
				httputil.DomainError(w, err)
				return
			}

			requestedTenantID := requestedTenantID(r)
			current, err := userAuthUC.GetCurrentUser(r.Context(), session.UserID, requestedTenantID)
			if err != nil {
				httputil.DomainError(w, err)
				return
			}
			if requestedTenantID != "" && (current.Tenant == nil || current.Tenant.ID != requestedTenantID) {
				httputil.DomainError(w, domain.ErrTenantAccessDenied)
				return
			}

			ctx := context.WithValue(r.Context(), userCtxKey{}, current.User.ID)
			ctx = context.WithValue(ctx, sessionCtxKey{}, session.ID)
			if current.Tenant != nil && current.Membership != nil {
				ctx = context.WithValue(ctx, tenantCtxKey{}, current.Tenant.ID)
				if instanceID := requestedInstanceID(r); instanceID != "" {
					ctx = context.WithValue(ctx, instanceCtxKey{}, instanceID)
				}
				ctx = context.WithValue(ctx, userRoleCtxKey{}, current.Membership.Role)
				ctx = logger.WithTenantID(ctx, current.Tenant.ID)
			}
			ctx = logger.WithUserID(ctx, current.User.ID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(userCtxKey{}).(string)
	return v
}

func UserSessionIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(sessionCtxKey{}).(string)
	return v
}

func UserRoleFromCtx(ctx context.Context) domain.UserRole {
	v, _ := ctx.Value(userRoleCtxKey{}).(domain.UserRole)
	return v
}

func RequireUserRole(allowed ...domain.UserRole) func(http.Handler) http.Handler {
	allowedSet := make(map[domain.UserRole]struct{}, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := UserRoleFromCtx(r.Context())
			if role == "" {
				httputil.DomainError(w, domain.ErrUnauthorized)
				return
			}
			if _, ok := allowedSet[role]; !ok {
				httputil.DomainError(w, domain.ErrUserRoleForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requestedTenantID(r *http.Request) string {
	if tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); tenantID != "" {
		return tenantID
	}
	return strings.TrimSpace(r.URL.Query().Get("tenant_id"))
}
