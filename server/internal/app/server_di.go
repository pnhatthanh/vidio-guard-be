package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/handlers"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/queue"
	"github.com/pnhatthanh/vidio-guard-be/internal/realtime"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
	"github.com/pnhatthanh/vidio-guard-be/internal/ws"
	"github.com/pnhatthanh/vidio-guard-be/internal/worker"
)

type container struct {
	store           pkg.StoreProvider
	cache           pkg.CacheProvider
	db              pkg.DBProvider
	enqueuer        queue.Enqueuer
	progressPublish *realtime.RedisPubSub
	progressSub     *realtime.RedisPubSub
}

func buildInfra(cfg *config.Config) (*container, error) {
	store, err := pkg.NewStoreProvider(
		cfg.Minio.Endpoint,
		cfg.Minio.PublicEndpoint,
		cfg.Minio.AccessKey,
		cfg.Minio.SecretKey,
		cfg.Minio.UseSSL,
		cfg.Minio.Bucket,
	)
	if err != nil {
		return nil, fmt.Errorf("init minio store: %w", err)
	}
	const maxRetries = 10
	const retryDelay = 3 * time.Second
	for i := 1; i <= maxRetries; i++ {
		if err := store.EnsureBucket(context.Background()); err == nil {
			log.Printf("[infra] bucket %q ready", cfg.Minio.Bucket)
			break
		} else if i == maxRetries {
			return nil, fmt.Errorf("ensure bucket after %d attempts: %w", maxRetries, err)
		} else {
			log.Printf("[infra] warn: ensure bucket (attempt %d/%d): %v — retrying in %s", i, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
		}
	}

	cache, err := pkg.NewCacheProvider(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return nil, fmt.Errorf("init redis cache: %w", err)
	}

	db, err := pkg.NewDBProvider(&cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres db: %w", err)
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

	progressPublish, err := realtime.NewRedisPubSub(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.ProgressChannel,
	)
	if err != nil {
		return nil, fmt.Errorf("init redis progress publisher: %w", err)
	}
	progressSub, err := realtime.NewRedisPubSub(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.ProgressChannel,
	)
	if err != nil {
		_ = progressPublish.Close()
		return nil, fmt.Errorf("init redis progress subscriber: %w", err)
	}

	return &container{
		store:           store,
		cache:           cache,
		db:              db,
		enqueuer:        enqueuer,
		progressPublish: progressPublish,
		progressSub:     progressSub,
	}, nil
}

func buildServer(cfg *config.Config, c *container) (*Server, error) {
	gdb := c.db.DB()

	userRepo := repository.NewUserRepository(gdb)
	tokenRepo := repository.NewTokenRepository(gdb)
	videoRepo := repository.NewVideoRepository(gdb)
	verdictRepo := repository.NewFinalVerdictRepository(gdb)
	violationRepo := repository.NewViolationSegmentRepository(gdb)

	tokenSvc := services.NewTokenService(&cfg.JWT, c.cache)
	mailer := pkg.NewMailer(cfg.SMTP)
	authSvc := services.NewAuthService(userRepo, tokenRepo, tokenSvc, c.cache, mailer, &cfg.Google, &cfg.JWT, cfg.PasswordReset)
	userSvc := services.NewUserService(userRepo, c.store, cfg.Minio.PresignURLTTL)
	videoSvc := services.NewVideoService(videoRepo, verdictRepo, violationRepo, c.enqueuer, c.store, cfg.Minio.PresignURLTTL)

	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	videoHandler := handlers.NewVideoHandler(videoSvc)

	hub := ws.NewHub()
	pipelineWS := ws.NewPipelineHandler(hub, tokenSvc)

	s := newServer(&cfg.Server)
	s.videoHandler = videoHandler
	s.userHandler = userHandler
	s.authHandler = authHandler
	s.tokenService = tokenSvc
	s.pipelineWS = pipelineWS
	s.hub = hub
	s.progressSub = c.progressSub
	s.progressPublish = c.progressPublish
	s.db = c.db
	s.registerMiddleware()
	s.registerRoutes()

	return s, nil
}

func buildWorker(cfg *config.Config, c *container) (*Worker, error) {
	w, err := NewWorker(&cfg.Redis, &cfg.Asynq)
	if err != nil {
		return nil, fmt.Errorf("init asynq worker: %w", err)
	}

	gdb := c.db.DB()
	videoRepo := repository.NewVideoRepository(gdb)
	verdictRepo := repository.NewFinalVerdictRepository(gdb)
	violationRepo := repository.NewViolationSegmentRepository(gdb)

	progress := services.NewVideoProgress(videoRepo, c.progressPublish)
	aiModerator := services.NewAIModerator(cfg.AIService)
	processor := services.NewFFmpegVideoProcessor(cfg.OutputDir, aiModerator)
	scorer := services.NewModerationScorer(cfg.Moderation)
	processingSvc := services.NewVideoProcessingService(
		videoRepo,
		verdictRepo,
		violationRepo,
		processor,
		c.store,
		progress,
		scorer,
		cfg.OutputDir,
	)

	videoHandler := &worker.VideoProcessHandler{Processing: processingSvc}
	w.RegisterHandler(worker.TypeVideoProcess, videoHandler.Handle)

	return w, nil
}
