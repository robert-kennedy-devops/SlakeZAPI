package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// ─── Request ID ──────────────────────────────────────────────────────────────

// RequestID generates a unique ID per request and stores it in context + header.
// A client-supplied X-Request-ID is accepted only if it is a valid UUID;
// any other value is silently replaced to prevent log injection.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !isValidRequestID(id) {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := logger.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isValidRequestID accepts only canonical UUID strings (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
func isValidRequestID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// ─── Logging ─────────────────────────────────────────────────────────────────

// Logger logs method, path, status and latency for every request.
func Logging(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			log.WithContext(r.Context()).Info("request", map[string]interface{}{
				"method":  r.Method,
				"path":    r.URL.Path,
				"status":  rw.status,
				"latency": time.Since(start).Milliseconds(),
			})
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// ─── Rate Limiter ─────────────────────────────────────────────────────────────

// bucket tracks a token bucket per key with idle-expiry for cleanup.
type bucket struct {
	mu       sync.Mutex
	tokens   float64
	last     time.Time
	lastUsed time.Time
	rps      float64
}

func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.lastUsed = now

	b.tokens += elapsed * b.rps
	if b.tokens > b.rps {
		b.tokens = b.rps
	}
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// newLimiter returns a bucket map with a background cleanup goroutine that
// evicts entries idle for more than 10 minutes, preventing unbounded growth.
func newLimiter(rps float64) (getBucket func(string) *bucket, stop func()) {
	var mu sync.Mutex
	buckets := make(map[string]*bucket)

	get := func(key string) *bucket {
		mu.Lock()
		defer mu.Unlock()
		if b, ok := buckets[key]; ok {
			return b
		}
		b := &bucket{tokens: rps, last: time.Now(), lastUsed: time.Now(), rps: rps}
		buckets[key] = b
		return b
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case now := <-ticker.C:
				mu.Lock()
				for k, b := range buckets {
					b.mu.Lock()
					idle := now.Sub(b.lastUsed)
					b.mu.Unlock()
					if idle > 10*time.Minute {
						delete(buckets, k)
					}
				}
				mu.Unlock()
			}
		}
	}()

	return get, func() { close(done) }
}

// RateLimit limits requests per second per tenant (identified via context).
// Falls back to client IP if no tenant is present.
func RateLimit(rps int) func(http.Handler) http.Handler {
	getBucket, _ := newLimiter(float64(rps))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := TenantFromCtx(r.Context())
			if key == "" {
				key = clientIP(r)
			}
			if !getBucket(key).allow() {
				httputil.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimit limits requests per minute by client IP.
// Intended for unauthenticated endpoints (login, signup, refresh).
func IPRateLimit(rpm int) func(http.Handler) http.Handler {
	rps := float64(rpm) / 60.0
	getBucket, _ := newLimiter(rps)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !getBucket(clientIP(r)).allow() {
				httputil.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the real client IP, preferring X-Real-IP set by a trusted
// reverse proxy, then stripping the port from RemoteAddr as fallback.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ─── Metrics Auth ─────────────────────────────────────────────────────────────

// MetricsAuth guards GET /metrics.
// If token is set, requires "Authorization: Bearer <token>".
// If token is empty, only loopback connections are allowed (local Prometheus).
func MetricsAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token != "" {
				got := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
				if got == "" || got != token {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			} else {
				ip := net.ParseIP(clientIP(r))
				if ip == nil || !ip.IsLoopback() {
					http.NotFound(w, r)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─── Bootstrap Auth ───────────────────────────────────────────────────────────

// BootstrapAuth protects the bootstrap endpoint with a pre-shared secret.
// If secret is empty the endpoint is disabled (returns 404).
func BootstrapAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				http.NotFound(w, r)
				return
			}
			got := strings.TrimSpace(r.Header.Get("X-Bootstrap-Secret"))
			if got == "" || got != secret {
				httputil.Error(w, http.StatusUnauthorized, "invalid bootstrap secret")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ─── Recover ─────────────────────────────────────────────────────────────────

// Recover catches panics and returns a 500.
func Recover(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.WithContext(r.Context()).Error("panic recovered", map[string]interface{}{
						"panic": rec,
					})
					httputil.Error(w, http.StatusInternalServerError, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS enables browser access for configured origins.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
		}
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || hasOrigin(allowed, origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, X-Tenant-ID, X-Instance-ID")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func hasOrigin(allowed map[string]struct{}, origin string) bool {
	_, ok := allowed[origin]
	return ok
}
