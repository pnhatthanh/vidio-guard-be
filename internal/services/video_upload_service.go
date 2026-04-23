package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
)

var (
	ErrUploadInvalidInput = errors.New("upload invalid input")
	ErrUploadStore        = errors.New("upload store failed")
	ErrUploadEnqueue      = errors.New("upload enqueue failed")
)

type VideoUploadService interface {
	Upload(ctx context.Context, reader io.Reader, size int64, originalFilename, contentType string) (videoID string, status constants.VideoStatus, err error)
}

type VideoProcessEnqueuer interface {
	EnqueueVideoProcess(videoID, objectKey string) error
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

func (s *videoUploadService) Upload(ctx context.Context, reader io.Reader, size int64, originalFilename, contentType string) (string, constants.VideoStatus, error) {
	if ctx == nil {
		return "", "", fmt.Errorf("%w: ctx is required", ErrUploadInvalidInput)
	}
	if reader == nil {
		return "", "", fmt.Errorf("%w: reader is required", ErrUploadInvalidInput)
	}
	if size <= 0 {
		return "", "", fmt.Errorf("%w: size must be > 0", ErrUploadInvalidInput)
	}

	videoID := uuid.NewString()

	ext := filepath.Ext(originalFilename)
	if ext == "" {
		ext = ".mp4"
	}
	objectKey := videoID + ext
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := s.store.Put(ctx, objectKey, reader, size, contentType); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrUploadStore, err)
	}
	if err := s.enqueuer.EnqueueVideoProcess(videoID, objectKey); err != nil {
		return videoID, constants.StatusFailed, fmt.Errorf("%w: %v", ErrUploadEnqueue, err)
	}

	return videoID, constants.StatusUploaded, nil
}
