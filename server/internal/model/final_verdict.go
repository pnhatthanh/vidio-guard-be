package model

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type FinalVerdict struct {
	ID                 uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	VideoID            uuid.UUID      `gorm:"type:uuid;column:video_id;not null;uniqueIndex"`
	Video              Video          `gorm:"foreignKey:VideoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Verdict            string         `gorm:"column:verdict;not null"`
	RiskScore          float64        `gorm:"column:risk_score;not null"`
	PeakViolenceScore  float64        `gorm:"column:peak_violence_score;not null"`
	PeakNsfwScore      float64        `gorm:"column:peak_nsfw_score;not null"`
	FlaggedFramesCount int            `gorm:"column:flagged_frames_count;not null"`
	FlaggedTimestamps  datatypes.JSON `gorm:"column:flagged_timestamps;type:jsonb;not null"`
}

func (FinalVerdict) TableName() string { return "final_verdicts" }
