package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"gorm.io/gorm"
)

type VideoRepository interface {
	Create(ctx context.Context, video *model.Video) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Video, error)
	ListByUser(ctx context.Context, p model.VideoListParams) ([]model.VideoListRow, int64, error)
	UpdateProgress(ctx context.Context, id uuid.UUID, status constants.VideoStatus, stage constants.VideoStage, percent int) error
	UpdateDuration(ctx context.Context, id uuid.UUID, durationSec float64) error
	MarkFailed(ctx context.Context, id uuid.UUID) error
	MarkCompleted(ctx context.Context, id uuid.UUID) error
	DeleteByID(ctx context.Context, id uuid.UUID) error
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
		"status":           status,
		"current_stage":    string(stage),
		"progress_percent": percent,
	}).Error
}

func (r *videoRepository) UpdateDuration(ctx context.Context, id uuid.UUID, durationSec float64) error {
	if durationSec <= 0 {
		return nil
	}
	sec := int(durationSec + 0.5)
	return r.db.WithContext(ctx).Model(&model.Video{}).Where("id = ?", id).Updates(map[string]any{
		"duration_seconds": sec,
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

func (r *videoRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Video{}).Error
}

func (r *videoRepository) ListByUser(ctx context.Context, p model.VideoListParams) ([]model.VideoListRow, int64, error) {
	countQ := r.db.WithContext(ctx).
		Table("videos v").
		Where("v.user_id = ?", p.UserID)
	if p.Filter == "violated" || p.Filter == "safe" {
		countQ = countQ.Joins("LEFT JOIN final_verdicts fv ON fv.video_id = v.id")
	}
	countQ = applyVideoListFilters(countQ, p)

	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := r.db.WithContext(ctx).
		Table("videos v").
		Select(`
			v.id,
			v.video_url,
			v.original_filename,
			v.file_size_bytes,
			v.status,
			v.progress_percent,
			v.current_stage,
			v.uploaded_at,
			v.processed_at,
			fv.verdict,
			fv.risk_score,
			COALESCE(vs.cnt, 0) AS violation_count`).
		Joins("LEFT JOIN final_verdicts fv ON fv.video_id = v.id").
		Joins(`LEFT JOIN (
			SELECT video_id, COUNT(*)::bigint AS cnt
			FROM violation_segments
			GROUP BY video_id
		) vs ON vs.video_id = v.id`).
		Where("v.user_id = ?", p.UserID)

	q = applyVideoListFilters(q, p)
	q = applyVideoListOrder(q, p)

	var rows []model.VideoListRow
	if err := q.Offset(p.Offset).Limit(p.Limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func applyVideoListFilters(q *gorm.DB, p model.VideoListParams) *gorm.DB {
	if s := strings.TrimSpace(p.Search); s != "" {
		q = q.Where("v.original_filename ILIKE ?", "%"+s+"%")
	}
	if p.Status != "" {
		q = q.Where("v.status = ?", p.Status)
	}
	switch p.Filter {
	case "violated":
		q = q.Where("fv.verdict IS NOT NULL AND fv.verdict <> ?", "safe")
	case "safe":
		q = q.Where("fv.verdict = ?", "safe")
	}
	if p.Since != nil {
		q = q.Where("COALESCE(v.processed_at, v.uploaded_at) >= ?", *p.Since)
	}
	return q
}

func applyVideoListOrder(q *gorm.DB, p model.VideoListParams) *gorm.DB {
	dir := "ASC"
	if p.Desc {
		dir = "DESC"
	}
	var col string
	switch p.Sort {
	case "uploaded_at":
		col = "v.uploaded_at"
	case "risk_score":
		col = "fv.risk_score"
	case "filename":
		col = "v.original_filename"
	default:
		col = "COALESCE(v.processed_at, v.uploaded_at)"
	}
	return q.Order(fmt.Sprintf("%s %s NULLS LAST", col, dir))
}
