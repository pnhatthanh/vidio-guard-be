package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/gorm"
)

type FinalVerdictRepository interface {
	Create(ctx context.Context, verdict *model.FinalVerdict) error
	FindByVideoID(ctx context.Context, videoID uuid.UUID) (*model.FinalVerdict, error)
	DeleteByVideoID(ctx context.Context, videoID uuid.UUID) error
}

type finalVerdictRepository struct {
	db *gorm.DB
}

func NewFinalVerdictRepository(db *gorm.DB) FinalVerdictRepository {
	return &finalVerdictRepository{db: db}
}

func (r *finalVerdictRepository) Create(ctx context.Context, verdict *model.FinalVerdict) error {
	return r.db.WithContext(ctx).Create(verdict).Error
}

func (r *finalVerdictRepository) FindByVideoID(ctx context.Context, videoID uuid.UUID) (*model.FinalVerdict, error) {
	var verdict model.FinalVerdict
	if err := r.db.WithContext(ctx).Where("video_id = ?", videoID).First(&verdict).Error; err != nil {
		return nil, err
	}
	return &verdict, nil
}

func (r *finalVerdictRepository) DeleteByVideoID(ctx context.Context, videoID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("video_id = ?", videoID).Delete(&model.FinalVerdict{}).Error
}
