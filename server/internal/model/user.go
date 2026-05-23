package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	FullName     string    `gorm:"column:full_name"`
	AvatarURL    *string   `gorm:"column:avatar_url"`
	Email        string    `gorm:"column:email"`
	PasswordHash string    `gorm:"column:password_hash"`
	GoogleID     *string   `gorm:"column:google_id"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`

	Videos        []Video        `gorm:"foreignKey:UserID;references:ID"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID;references:ID"`
}

func (User) TableName() string { return "users" }
