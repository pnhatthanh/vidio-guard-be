package services

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
	"gorm.io/gorm"
)

// VideoProcessEnqueuer enqueues a video for background processing.
type VideoProcessEnqueuer interface {
	EnqueueVideoProcess(videoID, objectKey string) error
}

// VideoService handles authenticated video upload and status queries.
type VideoService interface {
	Upload(ctx context.Context, userID uuid.UUID, reader io.Reader, size int64, filename, contentType string) (*dto.UploadVideoResponse, error)
	GetStatus(ctx context.Context, userID, videoID uuid.UUID) (*dto.VideoStatusResponse, error)
}

type videoService struct {
	videos   repository.VideoRepository
	verdicts repository.FinalVerdictRepository
	enqueuer VideoProcessEnqueuer
	store    pkg.StoreProvider
}

func NewVideoService(
	videos repository.VideoRepository,
	verdicts repository.FinalVerdictRepository,
	enqueuer VideoProcessEnqueuer,
	store pkg.StoreProvider,
) VideoService {
	return &videoService{
		videos:   videos,
		verdicts: verdicts,
		enqueuer: enqueuer,
		store:    store,
	}
}

func (s *videoService) Upload(
	ctx context.Context,
	userID uuid.UUID,
	reader io.Reader,
	size int64,
	filename, contentType string,
) (*dto.UploadVideoResponse, error) {
	if userID == uuid.Nil {
		return nil, apperror.NewUnauthorizedError("authentication required")
	}
	if size <= 0 {
		return nil, apperror.NewBadRequestError("file size must be greater than 0")
	}

	videoID := uuid.New()
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".mp4"
	}
	objectKey := videoID.String() + ext

	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	video := &model.Video{
		ID:               videoID,
		UserID:           userID,
		OriginalFilename: filepath.Base(filename),
		VideoURL:         objectKey,
		FileSizeBytes:    size,
		Status:           constants.StatusUploaded,
		ProgressPercent:  0,
		CurrentStage:     "",
	}

	if err := s.videos.Create(ctx, video); err != nil {
		return nil, apperror.NewInternalServerError("failed to create video record")
	}

	if err := s.store.Put(ctx, objectKey, reader, size, contentType); err != nil {
		_ = s.videos.MarkFailed(ctx, videoID)
		return nil, apperror.NewInternalServerError("failed to store video")
	}

	if err := s.videos.UpdateProgress(ctx, videoID, constants.StatusProcessing, constants.StageStarting, 0); err != nil {
		return nil, apperror.NewInternalServerError("failed to update video status")
	}

	if err := s.enqueuer.EnqueueVideoProcess(videoID.String(), objectKey); err != nil {
		_ = s.videos.MarkFailed(ctx, videoID)
		return nil, apperror.NewInternalServerError("failed to enqueue video processing")
	}

	return &dto.UploadVideoResponse{
		VideoID: videoID.String(),
		Status:  constants.StatusProcessing,
		Stage:   string(constants.StageStarting),
		ProgressPercent: 0,
	}, nil
}

func (s *videoService) GetStatus(ctx context.Context, userID, videoID uuid.UUID) (*dto.VideoStatusResponse, error) {
	video, err := s.videos.FindByIDAndUser(ctx, videoID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("video not found")
		}
		return nil, apperror.NewInternalServerError("failed to load video")
	}

	res := &dto.VideoStatusResponse{
		VideoID:         video.ID.String(),
		Status:          video.Status,
		Stage:           video.CurrentStage,
		ProgressPercent: video.ProgressPercent,
		OriginalFilename: video.OriginalFilename,
		UploadedAt:      video.UploadedAt,
		ProcessedAt:     video.ProcessedAt,
	}

	if video.Status == constants.StatusCompleted {
		verdict, err := s.verdicts.FindByVideoID(ctx, videoID)
		if err == nil && verdict != nil {
			res.Verdict = &dto.VideoVerdictSummary{
				Verdict:            verdict.Verdict,
				RiskScore:          verdict.RiskScore,
				PeakViolenceScore:  verdict.PeakViolenceScore,
				PeakNsfwScore:      verdict.PeakNsfwScore,
				FlaggedFramesCount: verdict.FlaggedFramesCount,
			}
		}
	}

	return res, nil
}
