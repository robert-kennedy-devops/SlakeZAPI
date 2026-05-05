package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/whatsapp-saas/api/internal/app"
	"github.com/whatsapp-saas/api/internal/config"
	"github.com/whatsapp-saas/api/pkg/logger"
)

func main() {
	log := logger.New()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	application, err := app.New(cfg, log)
	if err != nil {
		log.Error("failed to initialise application", map[string]interface{}{"err": err.Error()})
		os.Exit(1)
	}

	// Trap SIGINT / SIGTERM for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		log.Error("application error", map[string]interface{}{"err": err.Error()})
		os.Exit(1)
	}
}
