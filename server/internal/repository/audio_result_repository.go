package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/gorm"
)

type AudioResultRepository interface {
	Create(ctx context.Context, result *model.AudioResult) error
	DeleteByVideoID(ctx context.Context, videoID uuid.UUID) error
}

type audioResultRepository struct {
	db *gorm.DB
}

func NewAudioResultRepository(db *gorm.DB) AudioResultRepository {
	return &audioResultRepository{db: db}
}

func (r *audioResultRepository) Create(ctx context.Context, result *model.AudioResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

func (r *audioResultRepository) DeleteByVideoID(ctx context.Context, videoID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("video_id = ?", videoID).Delete(&model.AudioResult{}).Error
}
