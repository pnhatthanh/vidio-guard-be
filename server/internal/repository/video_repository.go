package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/gorm"
)

type VideoRepository interface {
	Create(ctx context.Context, video *model.Video) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Video, error)
	FindByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*model.Video, error)
	UpdateProgress(ctx context.Context, id uuid.UUID, status constants.VideoStatus, stage constants.VideoStage, percent int) error
	MarkFailed(ctx context.Context, id uuid.UUID) error
	MarkCompleted(ctx context.Context, id uuid.UUID) error
}

type videoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) VideoRepository {
	return &videoRepository{db: db}
}

func (r *videoRepository) Create(ctx context.Context, video *model.Video) error {
	return r.db.WithContext(ctx).Create(video).Error
}

func (r *videoRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Video, error) {
	var video model.Video
	if err := r.db.WithContext(ctx).First(&video, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *videoRepository) FindByIDAndUser(ctx context.Context, id, userID uuid.UUID) (*model.Video, error) {
	var video model.Video
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&video).Error; err != nil {
		return nil, err
	}
	return &video, nil
}

func (r *videoRepository) UpdateProgress(ctx context.Context, id uuid.UUID, status constants.VideoStatus, stage constants.VideoStage, percent int) error {
	return r.db.WithContext(ctx).Model(&model.Video{}).Where("id = ?", id).Updates(map[string]any{
		"status":            status,
		"current_stage":     string(stage),
		"progress_percent":  percent,
	}).Error
}

func (r *videoRepository) MarkFailed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Video{}).Where("id = ?", id).Updates(map[string]any{
		"status":           constants.StatusFailed,
		"current_stage":    string(constants.StageFailed),
		"progress_percent": 0,
	}).Error
}

func (r *videoRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Video{}).Where("id = ?", id).Updates(map[string]any{
		"status":           constants.StatusCompleted,
		"current_stage":    string(constants.StageCompleted),
		"progress_percent": constants.StageProgress[constants.StageCompleted],
		"processed_at":     &now,
	}).Error
}
