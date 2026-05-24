package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
)

func GetCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	raw, ok := c.Get("userID")
	if !ok {
		return uuid.Nil, apperror.NewUnauthorizedError("authentication required")
	}
	userID, ok := raw.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, apperror.NewUnauthorizedError("authentication required")
	}
	return userID, nil
}
