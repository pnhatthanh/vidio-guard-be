package model

import "github.com/google/uuid"

type FrameResult struct {
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	VideoID          uuid.UUID `gorm:"type:uuid;column:video_id;not null;uniqueIndex:uk_frame_results_video_frame"`
	Video            Video     `gorm:"foreignKey:VideoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	FrameNumber      int       `gorm:"column:frame_number;not null;uniqueIndex:uk_frame_results_video_frame"`
	FrameURL         string    `gorm:"column:frame_url"`
	TimestampSeconds float64   `gorm:"column:timestamp_seconds;not null"`
	ViolenceScore    float64   `gorm:"column:violence_score;not null"`
	NsfwScore        float64   `gorm:"column:nsfw_score;not null"`
	SafeScore        float64   `gorm:"column:safe_score;not null"`
	PredictedLabel   string    `gorm:"column:predicted_label;not null"`
}

func (FrameResult) TableName() string { return "frame_results" }
