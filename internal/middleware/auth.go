package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

type tenantCtxKey struct{}

// Auth validates the Bearer API key in every request.
func Auth(authUC *usecase.AuthUsecase, log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := extractBearer(r)
			if raw == "" {
				httputil.Error(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			key, err := authUC.ValidateAPIKey(r.Context(), raw)
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			// Inject tenant ID into context
			ctx := context.WithValue(r.Context(), tenantCtxKey{}, key.TenantID)
			ctx = logger.WithTenantID(ctx, key.TenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantFromCtx retrieves the tenant ID stored by the Auth middleware.
func TenantFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(tenantCtxKey{}).(string)
	return v
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return strings.TrimSpace(r.URL.Query().Get("access_token"))
}
