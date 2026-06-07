package model

import "github.com/google/uuid"

type FinalVerdict struct {
	ID                 uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey;column:id"`
	VideoID            uuid.UUID `gorm:"type:uuid;column:video_id;not null;uniqueIndex"`
	Video              Video     `gorm:"foreignKey:VideoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Verdict            string    `gorm:"column:verdict;not null"`
	RiskScore          float64   `gorm:"column:risk_score;not null"` 
	FrameScore         float64   `gorm:"column:frame_score;not null"`
	AudioScore         float64   `gorm:"column:audio_score;not null"`
	TotalFrames        int       `gorm:"column:total_frames;not null"`
	HardRuleTriggered  bool      `gorm:"column:hard_rule_triggered;not null"`
	HardRuleReason     string    `gorm:"column:hard_rule_reason;type:text;not null"`
}

func (FinalVerdict) TableName() string { return "final_verdicts" }
