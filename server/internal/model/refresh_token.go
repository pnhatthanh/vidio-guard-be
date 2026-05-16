package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	UserID    uuid.UUID `gorm:"type:uuid;column:user_id"`
	TokenHash string    `gorm:"column:token_hash"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	IsRevoked bool      `gorm:"column:is_revoked;default:false"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
