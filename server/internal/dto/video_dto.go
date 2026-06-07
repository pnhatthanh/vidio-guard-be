package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
)

type VideoDownloadResponse struct {
	VideoID          string `json:"video_id"`
	DownloadURL      string `json:"download_url"`
	Filename         string `json:"filename"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type UploadVideoResponse struct {
	VideoID         string                `json:"video_id"`
	VideoURL        string                `json:"video_url"`
	Status          constants.VideoStatus `json:"status"`
	Stage           string                `json:"stage"`
	ProgressPercent int                   `json:"progress_percent"`
}

type ProcessingOutput struct {
	Frames *PredictionResult
	Audio  *AudioResult
}

type VideoStatusResponse struct {
	VideoID           string                    `json:"video_id"`
	VideoURL          string                    `json:"video_url"`
	Status            constants.VideoStatus     `json:"status"`
	Stage             string                    `json:"stage"`
	ProgressPercent   int                       `json:"progress_percent"`
	OriginalFilename  string                    `json:"original_filename"`
	UploadedAt        time.Time                 `json:"uploaded_at"`
	ProcessedAt       *time.Time                `json:"processed_at,omitempty"`
	Verdict           *VideoVerdictSummary      `json:"verdict,omitempty"`
	ViolationSegments []ViolationSegmentSummary `json:"violation_segments,omitempty"`
}

type ViolationSegmentSummary struct {
	Source    string  `json:"source"`
	Category  string  `json:"category"`
	StartSec  float64 `json:"start_sec"`
	EndSec    float64 `json:"end_sec"`
	PeakScore float64 `json:"peak_score,omitempty"`
	Evidence  string  `json:"evidence,omitempty"`
}

type VideoVerdictSummary struct {
	Verdict           string  `json:"verdict"`
	Violated          bool    `json:"violated"`
	RiskScore         float64 `json:"risk_score"`
	FinalScore        float64 `json:"final_score"`
	FrameScore        float64 `json:"frame_score"`
	AudioScore        float64 `json:"audio_score"`
	TotalFrames       int     `json:"total_frames"`
	VideoDurationSec  float64 `json:"video_duration_sec"`
	HardRuleTriggered bool    `json:"hard_rule_triggered"`
	HardRuleReason    string  `json:"hard_rule_reason,omitempty"`
	Transcript        string  `json:"transcript,omitempty"`
}
type VideoJob struct {
	VideoID   uuid.UUID
	VideoPath string
	ObjectKey string
}

type ListVideosQuery struct {
	Search string 
	Status string 
	Filter string
	Days   int    // 0 = all time; 7, 30 = rolling window from now
	Sort   string // processed_at | uploaded_at | risk_score | filename
	Order  string // asc | desc
	Page   int
	Limit  int
}

type VideoListItem struct {
	VideoID          string                `json:"video_id"`
	VideoURL         string                `json:"video_url"`
	OriginalFilename string                `json:"original_filename"`
	Status           constants.VideoStatus `json:"status"`
	Stage            string                `json:"stage"`
	ProgressPercent  int                   `json:"progress_percent"`
	FileSizeBytes    int64                 `json:"file_size_bytes"`
	UploadedAt       time.Time             `json:"uploaded_at"`
	ProcessedAt      *time.Time            `json:"processed_at,omitempty"`
	Verdict          string                `json:"verdict,omitempty"`
	Violated         bool                  `json:"violated"`
	RiskScore        float64               `json:"risk_score,omitempty"`
	ViolationCount   int                   `json:"violation_count"`
}

type VideoListResponse struct {
	Items      []VideoListItem `json:"items"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"total_pages"`
}
