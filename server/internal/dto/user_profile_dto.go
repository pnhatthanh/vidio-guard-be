package dto

import (
	"time"

	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

type UserProfileResponse struct {
	ID           string    `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	HasPassword  bool      `json:"has_password"`
	HasGoogle    bool      `json:"has_google"`
	CreatedAt    time.Time `json:"created_at"`
}

type UpdateProfileRequest struct {
	FullName  string  `json:"full_name" binding:"required,min=2,max=100"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,max=500"`
}

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current_password" binding:"required"`
	NewPassword        string `json:"new_password" binding:"required,min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required,eqfield=NewPassword"`
}

func NewUserProfileResponse(user *model.User) *UserProfileResponse {
	if user == nil {
		return nil
	}
	avatar := ""
	if user.AvatarURL != nil {
		avatar = *user.AvatarURL
	}
	hasGoogle := user.GoogleID != nil && *user.GoogleID != ""
	return &UserProfileResponse{
		ID:          user.ID.String(),
		FullName:    user.FullName,
		Email:       user.Email,
		AvatarURL:   avatar,
		HasPassword: user.PasswordHash != "",
		HasGoogle:   hasGoogle,
		CreatedAt:   user.CreatedAt,
	}
}
