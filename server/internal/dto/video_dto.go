package dto

import (
	"time"

	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
)

type UploadVideoResponse struct {
	VideoID         string                `json:"video_id"`
	Status          constants.VideoStatus `json:"status"`
	Stage           string                `json:"stage"`
	ProgressPercent int                   `json:"progress_percent"`
}

type ProcessingOutput struct {
	FramesDir string
	Frames    *PredictionResult
	Audio     *AudioResult
}

type VideoStatusResponse struct {
	VideoID          string                `json:"video_id"`
	Status           constants.VideoStatus `json:"status"`
	Stage            string                `json:"stage"`
	ProgressPercent  int                   `json:"progress_percent"`
	OriginalFilename string                `json:"original_filename"`
	UploadedAt       time.Time             `json:"uploaded_at"`
	ProcessedAt      *time.Time            `json:"processed_at,omitempty"`
	Verdict          *VideoVerdictSummary  `json:"verdict,omitempty"`
}

type VideoVerdictSummary struct {
	Verdict            string  `json:"verdict"`
	RiskScore          float64 `json:"risk_score"`
	PeakViolenceScore  float64 `json:"peak_violence_score"`
	PeakNsfwScore      float64 `json:"peak_nsfw_score"`
	FlaggedFramesCount int     `json:"flagged_frames_count"`
}
