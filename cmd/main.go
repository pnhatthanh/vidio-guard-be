package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pnhatthanh/vidio-guard-be/internal/app"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to init app: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := application.Run(ctx); err != nil {
		log.Fatalf("app error: %v", err)
	}
}
