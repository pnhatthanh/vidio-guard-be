package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
)

// VideoProgress tracks and persists video processing progress in the database.
type VideoProgress struct {
	videos repository.VideoRepository
}

func NewVideoProgress(videos repository.VideoRepository) *VideoProgress {
	return &VideoProgress{videos: videos}
}

func (p *VideoProgress) Update(ctx context.Context, videoID uuid.UUID, stage constants.VideoStage) error {
	percent := constants.StageProgress[stage]
	return p.videos.UpdateProgress(ctx, videoID, constants.StatusProcessing, stage, percent)
}

func (p *VideoProgress) MarkFailed(ctx context.Context, videoID uuid.UUID) error {
	return p.videos.MarkFailed(ctx, videoID)
}

func (p *VideoProgress) MarkCompleted(ctx context.Context, videoID uuid.UUID) error {
	return p.videos.MarkCompleted(ctx, videoID)
}
