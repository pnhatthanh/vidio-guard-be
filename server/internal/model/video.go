package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
)

type Video struct {
	ID               uuid.UUID             `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	UserID           uuid.UUID             `gorm:"type:uuid;column:user_id;not null;index"`
	User             User                  `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	OriginalFilename string                `gorm:"column:original_filename;not null"`
	VideoURL         string                `gorm:"column:video_url;not null"`
	FileSizeBytes    int64                 `gorm:"column:file_size_bytes;not null"`
	DurationSeconds  *int                  `gorm:"column:duration_seconds"`
	Status           constants.VideoStatus `gorm:"column:status;not null"`
	ProgressPercent  int                   `gorm:"column:progress_percent;not null"`
	CurrentStage     string                `gorm:"column:current_stage;not null"`
	UploadedAt       time.Time             `gorm:"column:uploaded_at;autoCreateTime"`
	ProcessedAt      *time.Time            `gorm:"column:processed_at"`

	FinalVerdict *FinalVerdict      `gorm:"foreignKey:VideoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Violations   []ViolationSegment `gorm:"foreignKey:VideoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

}

func (Video) TableName() string { return "videos" }

type VideoListParams struct {
	UserID uuid.UUID
	Search string
	Status constants.VideoStatus 
	Filter string                
	Since  *time.Time          
	Sort   string
	Desc   bool
	Offset int
	Limit  int
}

type VideoListRow struct {
	ID               uuid.UUID
	VideoURL         string // object key in MinIO
	OriginalFilename string
	FileSizeBytes    int64
	Status           constants.VideoStatus
	ProgressPercent  int
	CurrentStage     string
	UploadedAt       time.Time
	ProcessedAt      *time.Time
	Verdict          *string
	RiskScore        *float64
	ViolationCount   int64
}
