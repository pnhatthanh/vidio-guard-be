package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
	"github.com/pnhatthanh/vidio-guard-be/internal/handlers"
)

type Server struct {
	cfg           *config.ServerConfig
	router        *gin.Engine
	uploadHandler handlers.UploadHandler
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
	return nil
}

func (s *Server) Addr() string {
	return s.cfg.Addr
}
