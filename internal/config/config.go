package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	HTTPPort         string
	ShutdownTimeout  time.Duration

	// Database
	DatabaseURL      string
	DBMaxConnections int

	// Auth
	APIKeySalt       string

	// WhatsApp
	WhatsAppDataPath string

	// Queue
	WorkerCount      int
	QueueBufferSize  int

	// Webhook
	WebhookTimeout   time.Duration
	WebhookRetries   int

	// Rate Limiting
	RateLimitRPS     int    // requests per second per tenant
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		ShutdownTimeout:  getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		DBMaxConnections: getIntEnv("DB_MAX_CONNECTIONS", 20),
		APIKeySalt:       getEnv("API_KEY_SALT", "change-me-in-production"),
		WhatsAppDataPath: getEnv("WHATSAPP_DATA_PATH", "./data/whatsapp"),
		WorkerCount:      getIntEnv("WORKER_COUNT", 10),
		QueueBufferSize:  getIntEnv("QUEUE_BUFFER_SIZE", 500),
		WebhookTimeout:   getDurationEnv("WEBHOOK_TIMEOUT", 10*time.Second),
		WebhookRetries:   getIntEnv("WEBHOOK_RETRIES", 3),
		RateLimitRPS:     getIntEnv("RATE_LIMIT_RPS", 10),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.APIKeySalt == "change-me-in-production" {
		fmt.Println("[WARN] API_KEY_SALT is set to default — change this in production!")
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
