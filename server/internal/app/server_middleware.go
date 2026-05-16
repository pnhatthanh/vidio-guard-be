package app

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
)

func (s *Server) registerMiddleware() {
	s.router.Use(s.CORSMiddleware())
	s.router.Use(s.RequestLogger())
	s.router.Use(s.ErrorHandler())
}

func (s *Server) ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		if appErr, ok := err.(*apperror.AppError); ok {
			c.JSON(appErr.Code, appErr)
			return
		}
		c.JSON(500, apperror.NewInternalServerError("internal server error"))
	}
}

func (s *Server) RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[request] %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Printf("[response] %s %s - %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}

func (s *Server) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func (s *Server) JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.tokenService == nil {
			c.Error(apperror.NewInternalServerError("auth not configured"))
			c.Abort()
			return
		}

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			c.Error(apperror.NewUnauthorizedError("missing bearer token"))
			c.Abort()
			return
		}

		tokenStr := strings.TrimSpace(authHeader[7:])
		if tokenStr == "" {
			c.Error(apperror.NewUnauthorizedError("missing bearer token"))
			c.Abort()
			return
		}

		userID, jti, expiresAt, err := s.tokenService.ValidateAccessToken(tokenStr)
		if err != nil {
			c.Error(apperror.NewUnauthorizedError("invalid token"))
			c.Abort()
			return
		}

		blacklisted, err := s.tokenService.IsBlacklisted(c.Request.Context(), jti)
		if err != nil {
			c.Error(apperror.NewInternalServerError("failed to validate token"))
			c.Abort()
			return
		}
		if blacklisted {
			c.Error(apperror.NewUnauthorizedError("token revoked"))
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("jti", jti)
		c.Set("expiresAt", expiresAt)

		c.Writer.Header().Set("Cache-Control", "no-store")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Next()
	}
}
