package model

import (
	"github.com/google/uuid"
	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
)

type ViolationSegment struct {
	ID        uuid.UUID                   `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	VideoID   uuid.UUID                   `gorm:"type:uuid;column:video_id;not null;index:idx_violation_segments_video_start,priority:1"`
	Video     Video                       `gorm:"foreignKey:VideoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Source    constants.VideoSource       `gorm:"column:source;not null"`
	Category  constants.ViolationCategory `gorm:"column:category;not null"`
	StartSec  float64                     `gorm:"column:start_sec;not null;index:idx_violation_segments_video_start,priority:2"`
	EndSec    float64                     `gorm:"column:end_sec;not null"`
	PeakScore float64                     `gorm:"column:peak_score;not null"`
	Evidence  string                      `gorm:"column:evidence;type:text"`
}

func (ViolationSegment) TableName() string { return "violation_segments" }
