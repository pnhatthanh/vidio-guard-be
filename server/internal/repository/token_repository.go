package repository

import (
	"context"

	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/gorm"
)

type TokenRepository interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*model.RefreshToken, error)
	DeleteByHash(ctx context.Context, hash string) error
}

type tokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *tokenRepository) FindByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *tokenRepository) DeleteByHash(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", hash).Delete(&model.RefreshToken{}).Error
}
