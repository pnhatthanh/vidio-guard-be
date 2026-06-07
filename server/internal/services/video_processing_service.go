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
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
)

type VideoProcessingService interface {
	Process(ctx context.Context, videoID uuid.UUID, objectKey string) error
}

type videoProcessingService struct {
	videos     repository.VideoRepository
	verdicts   repository.FinalVerdictRepository
	violations repository.ViolationSegmentRepository
	processor  VideoProcessor
	store      pkg.StoreProvider
	progress   *VideoProgress
	scorer     *ModerationScorer
	tempDir    string
}

func NewVideoProcessingService(
	videos repository.VideoRepository,
	verdicts repository.FinalVerdictRepository,
	violations repository.ViolationSegmentRepository,
	processor VideoProcessor,
	store pkg.StoreProvider,
	progress *VideoProgress,
	scorer *ModerationScorer,
	tempDir string,
) VideoProcessingService {
	return &videoProcessingService{
		videos:     videos,
		verdicts:   verdicts,
		violations: violations,
		processor:  processor,
		store:      store,
		progress:   progress,
		scorer:     scorer,
		tempDir:    tempDir,
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

	job := dto.VideoJob{
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

	if dur, err := utils.ProbeDurationSec(tmpPath); err == nil {
		_ = s.videos.UpdateDuration(ctx, videoID, dur)
	}

	return tmpPath, cleanup, nil
}

func (s *videoProcessingService) persistResults(ctx context.Context, videoID uuid.UUID, output *dto.ProcessingOutput) error {
	if output == nil {
		return nil
	}

	_ = s.verdicts.DeleteByVideoID(ctx, videoID)
	_ = s.violations.DeleteByVideoID(ctx, videoID)

	durationSec := 0.0
	if video, err := s.videos.FindByID(ctx, videoID); err == nil && video.DurationSeconds != nil {
		durationSec = float64(*video.DurationSeconds)
	}

	verdict := buildFinalVerdict(videoID, output.Frames, output.Audio, durationSec, s.scorer)
	if err := s.verdicts.Create(ctx, verdict); err != nil {
		return fmt.Errorf("save final verdict: %w", err)
	}

	segments := buildViolationSegments(videoID, output.Frames, output.Audio)
	if err := s.violations.CreateBatch(ctx, segments); err != nil {
		return fmt.Errorf("save violation segments: %w", err)
	}

	log.Printf("[processing] video=%s verdict=%s final=%.4f frame=%.4f audio=%.4f violations=%d hard_rule=%v",
		videoID, verdict.Verdict, verdict.RiskScore, verdict.FrameScore, verdict.AudioScore,
		len(segments), verdict.HardRuleTriggered)

	return nil
}
