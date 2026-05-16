package dto

import (
	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
)

type UserDTO struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

func NewUserDTO(user *model.User) *UserDTO {
	return &UserDTO{
		ID:       user.ID,
		FullName: user.FullName,
		Email:    user.Email,
	}
}
