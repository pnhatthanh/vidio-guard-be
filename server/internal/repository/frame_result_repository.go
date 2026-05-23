package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/gorm"
)

type FrameResultRepository interface {
	CreateBatch(ctx context.Context, results []model.FrameResult) error
	DeleteByVideoID(ctx context.Context, videoID uuid.UUID) error
}

type frameResultRepository struct {
	db *gorm.DB
}

func NewFrameResultRepository(db *gorm.DB) FrameResultRepository {
	return &frameResultRepository{db: db}
}

func (r *frameResultRepository) CreateBatch(ctx context.Context, results []model.FrameResult) error {
	if len(results) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(results, 100).Error
}

func (r *frameResultRepository) DeleteByVideoID(ctx context.Context, videoID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("video_id = ?", videoID).Delete(&model.FrameResult{}).Error
}
