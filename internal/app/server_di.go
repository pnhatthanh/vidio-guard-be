package app

import (
	"context"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/handlers"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/queue"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
	"github.com/pnhatthanh/vidio-guard-be/internal/worker"
)

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

type container struct {
	store    pkg.StoreProvider
	cache    pkg.CacheProvider
	enqueuer queue.Enqueuer
}

func buildInfra(cfg *config.Config) (*container, error) {
	store, err := pkg.NewStoreProvider(
		cfg.Minio.Endpoint,
		cfg.Minio.AccessKey,
		cfg.Minio.SecretKey,
		cfg.Minio.UseSSL,
		cfg.Minio.Bucket,
	)
	if err != nil {
		return nil, fmt.Errorf("init minio store: %w", err)
	}
	if err := store.EnsureBucket(context.Background()); err != nil {
		log.Printf("[infra] warn: ensure bucket: %v", err)
	}
	cache, err := pkg.NewCacheProvider(&cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis cache: %w", err)
	}
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	asynqClient := asynq.NewClient(redisOpt)
	enqueuer := queue.NewAsynqEnqueuer(
		asynqClient,
		cfg.Asynq.Queue,
		cfg.Asynq.MaxRetry,
		cfg.Asynq.TaskTimeout,
	)
	return &container{
		store:    store,
		cache:    cache,
		enqueuer: enqueuer,
	}, nil
}

func buildServer(cfg *config.Config, c *container) (*Server, error) {
	uploadHandler := handlers.NewUploadHandler(c.enqueuer, c.store)

	s := newServer(&cfg.Server)
	s.injectHandlers(uploadHandler)
	s.registerMiddleware()
	s.registerRoutes()

	return s, nil
}

func buildWorker(cfg *config.Config, c *container) (*Worker, error) {
	w, err := NewWorker(&cfg.Redis, &cfg.Asynq)
	if err != nil {
		return nil, fmt.Errorf("init asynq worker: %w", err)
	}
	aiModerator := services.NewAIModerator(cfg.AIService)
	processor := services.NewFFmpegVideoProcessor(cfg.OutputDir, aiModerator)
	videoHandler := &worker.VideoProcessHandler{
		Processor: processor,
		Store:     c.store,
		TempDir:   cfg.OutputDir,
	}
	w.RegisterHandler(worker.TypeVideoProcess, videoHandler.Handle)

	return w, nil
}

