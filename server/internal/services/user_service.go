package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
	"gorm.io/gorm"
)

type UserService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error
}

type userService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) UserService {
	return &userService{users: users}
}

func (s *userService) GetProfile(ctx context.Context, userID uuid.UUID) (*dto.UserProfileResponse, error) {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.NewUserProfileResponse(user), nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error) {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.FullName = strings.TrimSpace(req.FullName)
	if req.AvatarURL != nil {
		trimmed := strings.TrimSpace(*req.AvatarURL)
		if trimmed == "" {
			user.AvatarURL = nil
		} else {
			user.AvatarURL = &trimmed
		}
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, apperror.NewInternalServerError("failed to update profile")
	}
	return dto.NewUserProfileResponse(user), nil
}

func (s *userService) ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return err
	}

	if user.PasswordHash == "" {
		return apperror.NewBadRequestError("account uses Google sign-in; password cannot be changed")
	}

	if err := utils.ComparePassword(user.PasswordHash, req.CurrentPassword); err != nil {
		return apperror.NewUnauthorizedError("current password is incorrect")
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return apperror.NewInternalServerError("failed to hash password")
	}

	user.PasswordHash = hash
	if err := s.users.Update(ctx, user); err != nil {
		return apperror.NewInternalServerError("failed to update password")
	}
	return nil
}

func (s *userService) loadUser(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	if userID == uuid.Nil {
		return nil, apperror.NewUnauthorizedError("authentication required")
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("user not found")
		}
		return nil, apperror.NewInternalServerError("failed to load user")
	}
	return user, nil
}
