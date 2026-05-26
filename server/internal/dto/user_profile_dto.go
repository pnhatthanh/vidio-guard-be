package dto

import (
	"io"
	"time"

	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

type UserProfileResponse struct {
	ID          string    `json:"id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	HasPassword bool      `json:"has_password"`
	HasGoogle   bool      `json:"has_google"`
	CreatedAt   time.Time `json:"created_at"`
}

// UpdateProfileInput is parsed from multipart PATCH /users/me.
type UpdateProfileInput struct {
	FullName          string
	RemoveAvatar      bool
	HasAvatar         bool
	AvatarReader      io.Reader
	AvatarSize        int64
	AvatarFilename    string
	AvatarContentType string
}

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current_password" binding:"required"`
	NewPassword        string `json:"new_password" binding:"required,min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required,eqfield=NewPassword"`
}

func NewUserProfileResponse(user *model.User, avatarURL string) *UserProfileResponse {
	if user == nil {
		return nil
	}
	hasGoogle := user.GoogleID != nil && *user.GoogleID != ""
	return &UserProfileResponse{
		ID:          user.ID.String(),
		FullName:    user.FullName,
		Email:       user.Email,
		AvatarURL:   avatarURL,
		HasPassword: user.PasswordHash != "",
		HasGoogle:   hasGoogle,
		CreatedAt:   user.CreatedAt,
	}
}
