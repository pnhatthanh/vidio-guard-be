package utils

import (
	"time"

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

func GetJTI(c *gin.Context) (string, error) {
	raw, ok := c.Get("jti")
	if !ok {
		return "", apperror.NewUnauthorizedError("authentication required")
	}
	jti, ok := raw.(string)
	if !ok || jti == "" {
		return "", apperror.NewUnauthorizedError("authentication required")
	}
	return jti, nil
}

func GetExpiresAt(c *gin.Context) (time.Time, error) {
	raw, ok := c.Get("expiresAt")
	if !ok {
		return time.Time{}, apperror.NewUnauthorizedError("authentication required")
	}
	expiresAt, ok := raw.(int64)
	if !ok || expiresAt <= 0 {
		return time.Time{}, apperror.NewUnauthorizedError("authentication required")
	}
	return time.Unix(expiresAt, 0), nil
}
