package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/gorm"
)

type ViolationSegmentRepository interface {
	CreateBatch(ctx context.Context, segments []model.ViolationSegment) error
	DeleteByVideoID(ctx context.Context, videoID uuid.UUID) error
	FindByVideoID(ctx context.Context, videoID uuid.UUID) ([]model.ViolationSegment, error)
}

type violationSegmentRepository struct {
	db *gorm.DB
}

func NewViolationSegmentRepository(db *gorm.DB) ViolationSegmentRepository {
	return &violationSegmentRepository{db: db}
}

func (r *violationSegmentRepository) CreateBatch(ctx context.Context, segments []model.ViolationSegment) error {
	if len(segments) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(segments, 100).Error
}

func (r *violationSegmentRepository) DeleteByVideoID(ctx context.Context, videoID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("video_id = ?", videoID).Delete(&model.ViolationSegment{}).Error
}

func (r *violationSegmentRepository) FindByVideoID(ctx context.Context, videoID uuid.UUID) ([]model.ViolationSegment, error) {
	var rows []model.ViolationSegment
	err := r.db.WithContext(ctx).
		Where("video_id = ?", videoID).
		Order("start_sec ASC").
		Find(&rows).Error
	return rows, err
}
