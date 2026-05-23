package model

import "github.com/google/uuid"

type AudioResult struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	VideoID        uuid.UUID `gorm:"type:uuid;column:video_id;not null;uniqueIndex"`
	Video          Video     `gorm:"foreignKey:VideoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Transcript     string    `gorm:"column:transcript;type:text"`
	OffensiveScore float64   `gorm:"column:offensive_score;not null"`
	HateScore      float64   `gorm:"column:hate_score;not null"`
	CleanScore     float64   `gorm:"column:clean_score;not null"`
	PredictedLabel string    `gorm:"column:predicted_label;not null"`
}

func (AudioResult) TableName() string { return "audio_results" }
