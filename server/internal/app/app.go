package app

import (
	"context"
	"fmt"
	"log"

	"github.com/pnhatthanh/vidio-guard-be/internal/config"
)

type App struct {
	config *config.Config
	server *Server
	worker *Worker
}

func New(cfg *config.Config) (*App, error) {
	c, err := buildInfra(cfg)
	if err != nil {
		return nil, fmt.Errorf("build infra: %w", err)
	}

	server, err := buildServer(cfg, c)
	if err != nil {
		return nil, fmt.Errorf("build server: %w", err)
	}

	w, err := buildWorker(cfg, c)
	if err != nil {
		return nil, fmt.Errorf("build worker: %w", err)
	}

	return &App{config: cfg, server: server, worker: w}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		log.Printf("[worker] starting")
		if err := a.worker.Start(); err != nil {
			errCh <- fmt.Errorf("asynq worker: %w", err)
		}
	}()

	go func() {
		log.Printf("[http] listening on %s", a.server.Addr())
		if err := a.server.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("http server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		log.Printf("[app] shutdown requested")
		_ = a.Shutdown(context.Background())
		return nil
	case err := <-errCh:
		_ = a.Shutdown(context.Background())
		return err
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	var shutdownErr error
	if a.worker != nil {
		a.worker.Shutdown()
	}
	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			shutdownErr = err
		}
	}
	return shutdownErr
}
