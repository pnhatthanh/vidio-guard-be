package services

import (
	"context"
	"errors"
	"io"
	"log"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/repository"
	"gorm.io/gorm"
)

const (
	videoListDefaultLimit = 20
	videoListMaxLimit     = 100
)

type VideoProcessEnqueuer interface {
	EnqueueVideoProcess(videoID, objectKey string) error
}

type VideoService interface {
	Upload(ctx context.Context, userID uuid.UUID, reader io.Reader, size int64, filename, contentType string) (*dto.UploadVideoResponse, error)
	List(ctx context.Context, userID uuid.UUID, q dto.ListVideosQuery) (*dto.VideoListResponse, error)
	GetStatus(ctx context.Context, userID, videoID uuid.UUID) (*dto.VideoStatusResponse, error)
	GetDownloadURL(ctx context.Context, userID, videoID uuid.UUID) (*dto.VideoDownloadResponse, error)
	Delete(ctx context.Context, userID, videoID uuid.UUID) error
}

type videoService struct {
	videos     repository.VideoRepository
	verdicts   repository.FinalVerdictRepository
	violations repository.ViolationSegmentRepository
	enqueuer   VideoProcessEnqueuer
	store      pkg.StoreProvider
	presignTTL time.Duration
}

func NewVideoService(
	videos repository.VideoRepository,
	verdicts repository.FinalVerdictRepository,
	violations repository.ViolationSegmentRepository,
	enqueuer VideoProcessEnqueuer,
	store pkg.StoreProvider,
	presignTTL time.Duration,
) VideoService {
	if presignTTL <= 0 {
		presignTTL = time.Hour
	}
	return &videoService{
		videos:     videos,
		verdicts:   verdicts,
		violations: violations,
		enqueuer:   enqueuer,
		store:      store,
		presignTTL: presignTTL,
	}
}

func (s *videoService) presignedVideoURL(ctx context.Context, objectKey string) string {
	if objectKey == "" {
		return ""
	}
	url, err := s.store.PresignedGetURL(ctx, objectKey, s.presignTTL, &pkg.PresignOptions{Disposition: pkg.PresignInline})
	if err != nil {
		log.Printf("[video] presign object=%s: %v", objectKey, err)
		return ""
	}
	return url
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
		VideoID:         videoID.String(),
		VideoURL:        s.presignedVideoURL(ctx, objectKey),
		Status:          constants.StatusProcessing,
		Stage:           string(constants.StageStarting),
		ProgressPercent: 0,
	}, nil
}

func (s *videoService) GetStatus(ctx context.Context, userID, videoID uuid.UUID) (*dto.VideoStatusResponse, error) {
	video, err := s.videos.FindByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("video not found")
		}
		return nil, apperror.NewInternalServerError("failed to load video")
	}
	if video.UserID != userID {
		return nil, apperror.NewUnauthorizedError("access denied")
	}

	res := &dto.VideoStatusResponse{
		VideoID:          video.ID.String(),
		VideoURL:         s.presignedVideoURL(ctx, video.VideoURL),
		Status:           video.Status,
		Stage:            video.CurrentStage,
		ProgressPercent:  video.ProgressPercent,
		OriginalFilename: video.OriginalFilename,
		UploadedAt:       video.UploadedAt,
		ProcessedAt:      video.ProcessedAt,
	}

	if video.Status == constants.StatusCompleted {
		verdict, err := s.verdicts.FindByVideoID(ctx, videoID)
		if err == nil && verdict != nil {
			res.Verdict = &dto.VideoVerdictSummary{
				Verdict:           verdict.Verdict,
				Violated:          verdict.Verdict != "safe",
				RiskScore:         verdict.RiskScore,
				FrameScore:        verdict.FrameScore,
				AudioScore:        verdict.AudioScore,
				TotalFrames:       verdict.TotalFrames,
				VideoDurationSec:  verdict.VideoDurationSec,
				HardRuleTriggered: verdict.HardRuleTriggered,
				HardRuleReason:    verdict.HardRuleReason,
			}
		}
		segments, err := s.violations.FindByVideoID(ctx, videoID)
		if err == nil {
			res.ViolationSegments = mapViolationSegmentsToDTO(segments)
		}
	}

	return res, nil
}

func (s *videoService) GetDownloadURL(ctx context.Context, userID, videoID uuid.UUID) (*dto.VideoDownloadResponse, error) {
	video, err := s.videos.FindByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewNotFoundError("video not found")
		}
		return nil, apperror.NewInternalServerError("failed to load video")
	}

	downloadURL, err := s.store.PresignedGetURL(ctx, video.VideoURL, s.presignTTL, &pkg.PresignOptions{
		Disposition: pkg.PresignAttachment,
		Filename:    video.OriginalFilename,
	})
	if err != nil {
		return nil, apperror.NewInternalServerError("failed to generate download url")
	}

	return &dto.VideoDownloadResponse{
		VideoID:          video.ID.String(),
		DownloadURL:      downloadURL,
		Filename:         video.OriginalFilename,
		ExpiresInSeconds: int(s.presignTTL.Seconds()),
	}, nil
}

func (s *videoService) Delete(ctx context.Context, userID, videoID uuid.UUID) error {
	if userID == uuid.Nil {
		return apperror.NewUnauthorizedError("authentication required")
	}

	video, err := s.videos.FindByID(ctx, videoID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NewNotFoundError("video not found")
		}
		return apperror.NewInternalServerError("failed to load video")
	}
	if video.UserID != userID {
		return apperror.NewUnauthorizedError("access denied")
	}

	if video.VideoURL != "" {
		if err := s.store.Remove(ctx, video.VideoURL); err != nil {
			log.Printf("[video] delete object=%s: %v", video.VideoURL, err)
		}
	}

	if err := s.videos.DeleteByID(ctx, videoID); err != nil {
		return apperror.NewInternalServerError("failed to delete video")
	}
	return nil
}

func (s *videoService) List(ctx context.Context, userID uuid.UUID, q dto.ListVideosQuery) (*dto.VideoListResponse, error) {
	if userID == uuid.Nil {
		return nil, apperror.NewUnauthorizedError("authentication required")
	}

	params, err := normalizeListVideosQuery(userID, q)
	if err != nil {
		return nil, err
	}

	rows, total, err := s.videos.ListByUser(ctx, params)
	if err != nil {
		return nil, apperror.NewInternalServerError("failed to list videos")
	}

	items := make([]dto.VideoListItem, len(rows))
	for i, row := range rows {
		items[i] = s.mapVideoListRow(ctx, row)
	}

	totalPages := 0
	if params.Limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(params.Limit)))
	}

	return &dto.VideoListResponse{
		Items:      items,
		Total:      total,
		Page:       q.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

func normalizeListVideosQuery(userID uuid.UUID, q dto.ListVideosQuery) (model.VideoListParams, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	limit := q.Limit
	if limit <= 0 {
		limit = videoListDefaultLimit
	}
	if limit > videoListMaxLimit {
		limit = videoListMaxLimit
	}

	sort := strings.ToLower(strings.TrimSpace(q.Sort))
	if sort == "" {
		sort = "processed_at"
	}
	switch sort {
	case "processed_at", "uploaded_at", "risk_score", "filename":
	default:
		return model.VideoListParams{}, apperror.NewBadRequestError(
			"sort must be one of: processed_at, uploaded_at, risk_score, filename",
		)
	}

	order := strings.ToLower(strings.TrimSpace(q.Order))
	if order == "" {
		order = "desc"
	}
	desc := true
	switch order {
	case "desc":
	case "asc":
		desc = false
	default:
		return model.VideoListParams{}, apperror.NewBadRequestError("order must be asc or desc")
	}

	status := constants.VideoStatus(strings.TrimSpace(q.Status))
	if status == "" || status == "all" {
		status = ""
	} else {
		switch status {
		case constants.StatusUploaded, constants.StatusProcessing, constants.StatusCompleted, constants.StatusFailed:
		default:
			return model.VideoListParams{}, apperror.NewBadRequestError(
				"status must be one of: all, uploaded, processing, completed, failed",
			)
		}
	}

	filter := strings.ToLower(strings.TrimSpace(q.Filter))
	if filter == "" {
		filter = "all"
	}
	switch filter {
	case "all", "violated", "safe":
	default:
		return model.VideoListParams{}, apperror.NewBadRequestError(
			"filter must be one of: all, violated, safe",
		)
	}

	var since *time.Time
	if q.Days > 0 {
		t := time.Now().AddDate(0, 0, -q.Days)
		since = &t
	}

	return model.VideoListParams{
		UserID: userID,
		Search: strings.TrimSpace(q.Search),
		Status: status,
		Filter: filter,
		Since:  since,
		Sort:   sort,
		Desc:   desc,
		Offset: (q.Page - 1) * limit,
		Limit:  limit,
	}, nil
}

func (s *videoService) mapVideoListRow(ctx context.Context, row model.VideoListRow) dto.VideoListItem {
	item := dto.VideoListItem{
		VideoID:          row.ID.String(),
		VideoURL:         s.presignedVideoURL(ctx, row.VideoURL),
		OriginalFilename: row.OriginalFilename,
		Status:           row.Status,
		Stage:            row.CurrentStage,
		ProgressPercent:  row.ProgressPercent,
		FileSizeBytes:    row.FileSizeBytes,
		UploadedAt:       row.UploadedAt,
		ProcessedAt:      row.ProcessedAt,
		ViolationCount:   int(row.ViolationCount),
	}
	if row.Verdict != nil {
		item.Verdict = *row.Verdict
		item.Violated = *row.Verdict != "safe"
	}
	if row.RiskScore != nil {
		item.RiskScore = *row.RiskScore
	}
	return item
}
