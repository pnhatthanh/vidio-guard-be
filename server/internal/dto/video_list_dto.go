package dto

import (
	"time"

	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
)

// ListVideosQuery is built from GET /videos query parameters.
type ListVideosQuery struct {
	Search string // filename ILIKE
	Status string // all | uploaded | processing | completed | failed
	Filter string // all | violated | safe — moderation outcome (completed videos)
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
