package services

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/realtime"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
)

type VideoProgress struct {
	videos    repository.VideoRepository
	publisher realtime.ProgressPublisher
}

func NewVideoProgress(videos repository.VideoRepository, publisher realtime.ProgressPublisher) *VideoProgress {
	return &VideoProgress{videos: videos, publisher: publisher}
}

func (p *VideoProgress) Update(ctx context.Context, videoID uuid.UUID, stage constants.VideoStage) error {
	percent := constants.StageProgress[stage]
	if err := p.videos.UpdateProgress(ctx, videoID, constants.StatusProcessing, stage, percent); err != nil {
		return err
	}
	p.notify(ctx, videoID, constants.StatusProcessing, stage, percent)
	return nil
}

func (p *VideoProgress) MarkFailed(ctx context.Context, videoID uuid.UUID) error {
	if err := p.videos.MarkFailed(ctx, videoID); err != nil {
		return err
	}
	p.notify(ctx, videoID, constants.StatusFailed, constants.StageFailed, 0)
	return nil
}

func (p *VideoProgress) MarkCompleted(ctx context.Context, videoID uuid.UUID) error {
	if err := p.videos.MarkCompleted(ctx, videoID); err != nil {
		return err
	}
	p.notify(ctx, videoID, constants.StatusCompleted, constants.StageCompleted, constants.StageProgress[constants.StageCompleted])
	return nil
}

func (p *VideoProgress) notify(ctx context.Context, videoID uuid.UUID, status constants.VideoStatus, stage constants.VideoStage, percent int) {
	if p.publisher == nil {
		return
	}
	video, err := p.videos.FindByID(ctx, videoID)
	if err != nil {
		log.Printf("[progress] skip publish video=%s: %v", videoID, err)
		return
	}
	ev := realtime.NewProgressEvent(
		video.UserID.String(),
		videoID.String(),
		status,
		string(stage),
		percent,
	)
	if err := p.publisher.Publish(ctx, ev); err != nil {
		log.Printf("[progress] publish video=%s: %v", videoID, err)
	}
}
