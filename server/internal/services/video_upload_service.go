package services

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
)

type VideoProcessEnqueuer interface {
	EnqueueVideoProcess(videoID, objectKey string) error
}

type VideoUploadService interface {
	Upload(ctx context.Context, reader io.Reader, size int64, filename, contentType string) (*dto.UploadVideoResponse, error)
}

type videoUploadService struct {
	enqueuer VideoProcessEnqueuer
	store    pkg.StoreProvider
}

func NewVideoUploadService(enqueuer VideoProcessEnqueuer, store pkg.StoreProvider) VideoUploadService {
	return &videoUploadService{
		enqueuer: enqueuer,
		store:    store,
	}
}

func (s *videoUploadService) Upload(
	ctx context.Context,
	reader io.Reader,
	size int64,
	filename, contentType string,
) (*dto.UploadVideoResponse, error) {
	if size <= 0 {
		return nil, apperror.NewBadRequestError("file size must be greater than 0")
	}

	videoID := uuid.NewString()
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".mp4"
	}
	objectKey := videoID + ext

	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := s.store.Put(ctx, objectKey, reader, size, contentType); err != nil {
		return nil, apperror.NewInternalServerError("failed to store video")
	}

	if err := s.enqueuer.EnqueueVideoProcess(videoID, objectKey); err != nil {
		return &dto.UploadVideoResponse{
			VideoID: videoID,
			Status:  constants.StatusFailed,
		}, apperror.NewInternalServerError("failed to enqueue video processing")
	}

	return &dto.UploadVideoResponse{
		VideoID: videoID,
		Status:  constants.StatusUploaded,
	}, nil
}
