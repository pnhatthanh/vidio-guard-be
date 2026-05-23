package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
)

type VideoProcessingService interface {
	Process(ctx context.Context, videoID uuid.UUID, objectKey string) error
}

type videoProcessingService struct {
	videos    repository.VideoRepository
	frames    repository.FrameResultRepository
	audio     repository.AudioResultRepository
	verdicts  repository.FinalVerdictRepository
	processor VideoProcessor
	store     pkg.StoreProvider
	progress  *VideoProgress
	tempDir   string
}

func NewVideoProcessingService(
	videos repository.VideoRepository,
	frames repository.FrameResultRepository,
	audio repository.AudioResultRepository,
	verdicts repository.FinalVerdictRepository,
	processor VideoProcessor,
	store pkg.StoreProvider,
	progress *VideoProgress,
	tempDir string,
) VideoProcessingService {
	return &videoProcessingService{
		videos:    videos,
		frames:    frames,
		audio:     audio,
		verdicts:  verdicts,
		processor: processor,
		store:     store,
		progress:  progress,
		tempDir:   tempDir,
	}
}

func (s *videoProcessingService) Process(ctx context.Context, videoID uuid.UUID, objectKey string) error {
	if err := s.progress.Update(ctx, videoID, constants.StageStarting); err != nil {
		return err
	}

	tmpPath, cleanup, err := s.downloadVideo(ctx, videoID, objectKey)
	if err != nil {
		_ = s.progress.MarkFailed(ctx, videoID)
		return err
	}
	defer cleanup()

	job := model.VideoJob{
		VideoID:   videoID,
		VideoPath: tmpPath,
		ObjectKey: objectKey,
	}

	output, err := s.processor.Process(ctx, job, s.progress)
	if err != nil {
		_ = s.progress.MarkFailed(ctx, videoID)
		return err
	}

	if err := s.progress.Update(ctx, videoID, constants.StageAggregation); err != nil {
		return err
	}
	if err := s.persistResults(ctx, videoID, output); err != nil {
		_ = s.progress.MarkFailed(ctx, videoID)
		return err
	}

	return s.progress.MarkCompleted(ctx, videoID)
}

func (s *videoProcessingService) downloadVideo(ctx context.Context, videoID uuid.UUID, objectKey string) (string, func(), error) {
	ext := filepath.Ext(objectKey)
	tmpFile, err := os.CreateTemp(s.tempDir, "video-"+videoID.String()+"-")
	if err != nil {
		return "", nil, err
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	if ext != "" {
		newPath := tmpPath + ext
		if err := os.Rename(tmpPath, newPath); err != nil {
			_ = os.Remove(tmpPath)
			return "", nil, err
		}
		tmpPath = newPath
	}

	cleanup := func() { _ = os.Remove(tmpPath) }

	if err := s.store.DownloadToFile(ctx, objectKey, tmpPath); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("download object: %w", err)
	}
	return tmpPath, cleanup, nil
}

func (s *videoProcessingService) persistResults(ctx context.Context, videoID uuid.UUID, output *dto.ProcessingOutput) error {
	if output == nil {
		return nil
	}

	_ = s.frames.DeleteByVideoID(ctx, videoID)
	_ = s.audio.DeleteByVideoID(ctx, videoID)
	_ = s.verdicts.DeleteByVideoID(ctx, videoID)

	frameRows := mapFrameResults(videoID, output.FramesDir, output.Frames)
	if err := s.frames.CreateBatch(ctx, frameRows); err != nil {
		return fmt.Errorf("save frame results: %w", err)
	}

	if row := mapAudioResult(videoID, output.Audio); row != nil {
		if err := s.audio.Create(ctx, row); err != nil {
			return fmt.Errorf("save audio result: %w", err)
		}
	}

	verdict := buildFinalVerdict(videoID, output.Frames, output.Audio)
	if err := s.verdicts.Create(ctx, verdict); err != nil {
		return fmt.Errorf("save final verdict: %w", err)
	}

	log.Printf("[processing] video=%s verdict=%s risk=%.4f frames=%d",
		videoID, verdict.Verdict, verdict.RiskScore, len(frameRows))

	return nil
}
