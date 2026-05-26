package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
	"gorm.io/gorm"
)

const (
	avatarMaxBytes    = 5 << 20 // 5 MB
	avatarObjectPrefix = "avatars/"
)

var allowedAvatarContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type UserService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateProfileInput) (*dto.UserProfileResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error
}

type userService struct {
	users      repository.UserRepository
	store      pkg.StoreProvider
	presignTTL time.Duration
}

func NewUserService(users repository.UserRepository, store pkg.StoreProvider, presignTTL time.Duration) UserService {
	if presignTTL <= 0 {
		presignTTL = time.Hour
	}
	return &userService{
		users:      users,
		store:      store,
		presignTTL: presignTTL,
	}
}

func (s *userService) GetProfile(ctx context.Context, userID uuid.UUID) (*dto.UserProfileResponse, error) {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.NewUserProfileResponse(user, s.resolveAvatarURL(ctx, user)), nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateProfileInput) (*dto.UserProfileResponse, error) {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	fullName := strings.TrimSpace(input.FullName)
	if len(fullName) < 2 {
		return nil, apperror.NewBadRequestError("full_name must be at least 2 characters")
	}
	user.FullName = fullName

	oldAvatarKey := ""
	if user.AvatarURL != nil && !isExternalAvatarURL(*user.AvatarURL) {
		oldAvatarKey = *user.AvatarURL
	}

	switch {
	case input.HasAvatar:
		objectKey, err := s.uploadAvatar(ctx, userID, input.AvatarReader, input.AvatarSize, input.AvatarContentType)
		if err != nil {
			return nil, err
		}
		user.AvatarURL = &objectKey
		if oldAvatarKey != "" && oldAvatarKey != objectKey {
			_ = s.store.Remove(ctx, oldAvatarKey)
		}
	case input.RemoveAvatar:
		user.AvatarURL = nil
		if oldAvatarKey != "" {
			_ = s.store.Remove(ctx, oldAvatarKey)
		}
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, apperror.NewInternalServerError("failed to update profile")
	}
	return dto.NewUserProfileResponse(user, s.resolveAvatarURL(ctx, user)), nil
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

func (s *userService) uploadAvatar(
	ctx context.Context,
	userID uuid.UUID,
	reader io.Reader,
	size int64,
	contentType string,
) (string, error) {
	if size <= 0 {
		return "", apperror.NewBadRequestError("avatar file size must be greater than 0")
	}
	if size > avatarMaxBytes {
		return "", apperror.NewBadRequestError("avatar must be at most 5 MB")
	}

	contentType = strings.TrimSpace(strings.ToLower(contentType))
	ext, ok := allowedAvatarContentTypes[contentType]
	if !ok {
		return "", apperror.NewBadRequestError("avatar must be JPEG, PNG, or WebP")
	}

	objectKey := fmt.Sprintf("%s%s%s", avatarObjectPrefix, userID.String(), ext)
	if err := s.store.Put(ctx, objectKey, reader, size, contentType); err != nil {
		return "", apperror.NewInternalServerError("failed to store avatar")
	}
	return objectKey, nil
}

func (s *userService) resolveAvatarURL(ctx context.Context, user *model.User) string {
	if user == nil || user.AvatarURL == nil {
		return ""
	}
	key := strings.TrimSpace(*user.AvatarURL)
	if key == "" {
		return ""
	}
	if isExternalAvatarURL(key) {
		return key
	}
	url, err := s.store.PresignedGetURL(ctx, key, s.presignTTL, &pkg.PresignOptions{Disposition: pkg.PresignInline})
	if err != nil {
		log.Printf("[user] presign avatar=%s: %v", key, err)
		return ""
	}
	return url
}

func isExternalAvatarURL(value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")
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
