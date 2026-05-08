package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Server
	HTTPPort        string
	ShutdownTimeout time.Duration

	// Database
	DatabaseURL      string
	DBMaxConnections int

	// Auth
	BootstrapSecret           string // empty = bootstrap endpoint disabled
	APIKeySalt                string
	UserAccessTokenTTL        time.Duration
	UserRefreshTokenTTL       time.Duration
	UserSessionCookieName     string
	UserSessionCookieSecure   bool
	UserSessionCookieDomain   string
	UserSessionCookieSameSite string

	// WhatsApp
	WhatsAppDataPath string

	// Queue
	WorkerCount     int
	QueueBufferSize int

	// Webhook
	WebhookTimeout time.Duration
	WebhookRetries int

	// Rate Limiting
	RateLimitRPS     int // requests per second per tenant
	AuthRateLimitRPM int // requests per minute per IP for auth endpoints

	// Metrics
	MetricsToken string // if set, GET /metrics requires Authorization: Bearer <token>

	// CORS
	CORSAllowedOrigins []string

	// AI
	OpenAIAPIKey             string
	OpenAITranscriptionModel string

	// Billing / Stripe
	AppBaseURL            string
	StripeSecretKey       string
	StripeWebhookSecret   string
	StripePriceStarterID  string
	StripePriceGrowthID   string
	StripePriceProID      string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:                  getEnv("HTTP_PORT", "8080"),
		ShutdownTimeout:           getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		DatabaseURL:               getEnv("DATABASE_URL", ""),
		DBMaxConnections:          getIntEnv("DB_MAX_CONNECTIONS", 20),
		BootstrapSecret:           getEnv("BOOTSTRAP_SECRET", ""),
		APIKeySalt:                getEnv("API_KEY_SALT", "change-me-in-production"),
		UserAccessTokenTTL:        getDurationEnv("USER_ACCESS_TOKEN_TTL", 15*time.Minute),
		UserRefreshTokenTTL:       getDurationEnv("USER_REFRESH_TOKEN_TTL", 7*24*time.Hour),
		UserSessionCookieName:     getEnv("USER_SESSION_COOKIE_NAME", "slakezapi_rt"),
		UserSessionCookieSecure:   getBoolEnv("USER_SESSION_COOKIE_SECURE", false),
		UserSessionCookieDomain:   getEnv("USER_SESSION_COOKIE_DOMAIN", ""),
		UserSessionCookieSameSite: strings.ToLower(getEnv("USER_SESSION_COOKIE_SAMESITE", "lax")),
		WhatsAppDataPath:          getEnv("WHATSAPP_DATA_PATH", "./data/whatsapp"),
		WorkerCount:               getIntEnv("WORKER_COUNT", 10),
		QueueBufferSize:           getIntEnv("QUEUE_BUFFER_SIZE", 500),
		WebhookTimeout:            getDurationEnv("WEBHOOK_TIMEOUT", 10*time.Second),
		WebhookRetries:            getIntEnv("WEBHOOK_RETRIES", 3),
		RateLimitRPS:              getIntEnv("RATE_LIMIT_RPS", 10),
		AuthRateLimitRPM:          getIntEnv("AUTH_RATE_LIMIT_RPM", 20),
		MetricsToken:              getEnv("METRICS_TOKEN", ""),
		OpenAIAPIKey:              getEnv("OPENAI_API_KEY", ""),
		OpenAITranscriptionModel:  getEnv("OPENAI_TRANSCRIPTION_MODEL", "gpt-4o-mini-transcribe"),
		AppBaseURL:                getEnv("APP_BASE_URL", "http://localhost:3000"),
		StripeSecretKey:           getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:       getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePriceStarterID:      getEnv("STRIPE_PRICE_STARTER_ID", ""),
		StripePriceGrowthID:       getEnv("STRIPE_PRICE_GROWTH_ID", ""),
		StripePriceProID:          getEnv("STRIPE_PRICE_PRO_ID", ""),
		CORSAllowedOrigins: getListEnv("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:3000",
		}),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.APIKeySalt == "change-me-in-production" {
		fmt.Println("[WARN] API_KEY_SALT is set to default — change this in production!")
	}
	switch cfg.UserSessionCookieSameSite {
	case "lax", "strict", "none":
	default:
		return nil, fmt.Errorf("USER_SESSION_COOKIE_SAMESITE must be one of: lax, strict, none")
	}
	if cfg.UserSessionCookieSameSite == "none" && !cfg.UserSessionCookieSecure {
		cfg.UserSessionCookieSecure = true
		fmt.Println("[WARN] USER_SESSION_COOKIE_SAMESITE=none requires secure cookies — enabling USER_SESSION_COOKIE_SECURE automatically")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getBoolEnv(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getListEnv(key string, fallback []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
