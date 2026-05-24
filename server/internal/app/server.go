package app

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/handlers"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/realtime"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
	"github.com/pnhatthanh/vidio-guard-be/internal/ws"
)

type Server struct {
	cfg           *config.ServerConfig
	router        *gin.Engine
	videoHandler  handlers.VideoHandler
	userHandler   handlers.UserHandler
	authHandler   handlers.AuthHandler
	pipelineWS    *ws.PipelineHandler
	tokenService  services.TokenService
	hub           *ws.Hub
	progressSub     *realtime.RedisPubSub
	progressPublish *realtime.RedisPubSub
	db            pkg.DBProvider
}

func newServer(cfg *config.ServerConfig) *Server {
	return &Server{
		cfg:    cfg,
		router: gin.Default(),
	}
}

func (s *Server) ListenAndServe() error {
	return s.router.Run(s.cfg.Addr)
}

func (s *Server) RunRealtime(ctx context.Context) {
	if s.hub == nil || s.progressSub == nil {
		return
	}
	go s.hub.Run(ctx)
	go func() {
		if err := s.progressSub.Subscribe(ctx, s.hub.BroadcastProgress); err != nil && ctx.Err() == nil {
			log.Printf("[realtime] redis subscribe stopped: %v", err)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.progressSub != nil {
		_ = s.progressSub.Close()
	}
	if s.progressPublish != nil {
		_ = s.progressPublish.Close()
	}
	if s != nil && s.db != nil {
		_ = s.db.Close()
	}
	return nil
}

func (s *Server) Addr() string {
	return s.cfg.Addr
}
