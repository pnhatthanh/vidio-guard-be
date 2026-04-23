package app

import (
	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
)

func (s *Server) registerMiddleware() {
	s.router.Use(s.ErrorHandler())
}

func (s *Server) ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			if appErr, ok := err.(*apperror.AppError); ok {
				c.JSON(appErr.Code, appErr)
			} else {
				c.JSON(500, apperror.NewInternalServerError("internal server error"))
			}
		}
	}
}
func (s *Server) RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log request details here (e.g., method, path, headers)
		c.Next()
	}
}

