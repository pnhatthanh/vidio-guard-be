package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/handlers"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
)

type Server struct {
	cfg           *config.ServerConfig
	router        *gin.Engine
	videoHandler  handlers.VideoHandler
	authHandler   handlers.AuthHandler
	tokenService  services.TokenService
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

func (s *Server) Shutdown(ctx context.Context) error {
	if s != nil && s.db != nil {
		_ = s.db.Close()
	}
	return nil
}

func (s *Server) Addr() string {
	return s.cfg.Addr
}
