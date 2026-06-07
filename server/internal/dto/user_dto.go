package dto

import (
	"io"
	"time"

	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current_password" binding:"required"`
	NewPassword        string `json:"new_password" binding:"required,min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required,eqfield=NewPassword"`
}

type UserProfileResponse struct {
	ID          string    `json:"id"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	HasPassword bool      `json:"has_password"`
	HasGoogle   bool      `json:"has_google"`
	CreatedAt   time.Time `json:"created_at"`
}

type UpdateProfileInput struct {
	FullName          string    `form:"full_name"`
	RemoveAvatar      bool      `form:"remove_avatar"`
	HasAvatar         bool      `json:"has_avatar"`
	AvatarReader      io.Reader `json:"-"`
	AvatarSize        int64     `json:"-"`
	AvatarFilename    string    `json:"-"`
	AvatarContentType string    `json:"-"`
}

func NewUserProfileResponse(user *model.User) *UserProfileResponse {
	if user == nil {
		return nil
	}
	avatarURL := ""
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		avatarURL = *user.AvatarURL
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
