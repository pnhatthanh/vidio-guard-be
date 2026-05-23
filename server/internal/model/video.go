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

	AudioResult  *AudioResult  `gorm:"foreignKey:VideoID;references:ID"`
	FrameResults []FrameResult `gorm:"foreignKey:VideoID;references:ID"`
	FinalVerdict *FinalVerdict `gorm:"foreignKey:VideoID;references:ID"`
}

func (Video) TableName() string { return "videos" }
