package realtime

import (
	"encoding/json"

	"github.com/pnhatthanh/vidio-guard-be/internal/constants"
)

const MessageTypeProgress = "video.progress"

type ProgressEvent struct {
	Type            string                `json:"type"`
	UserID          string                `json:"user_id"`
	VideoID         string                `json:"video_id"`
	Status          constants.VideoStatus `json:"status"`
	Stage           constants.VideoStage  `json:"stage"`
	ProgressPercent int                   `json:"progress_percent"`
}

func NewProgressEvent(userID, videoID string, status constants.VideoStatus, stage constants.VideoStage, percent int) ProgressEvent {
	return ProgressEvent{
		Type:            MessageTypeProgress,
		UserID:          userID,
		VideoID:         videoID,
		Status:          status,
		Stage:           stage,
		ProgressPercent: percent,
	}
}

func (e ProgressEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func ParseProgressEvent(data []byte) (ProgressEvent, error) {
	var ev ProgressEvent
	err := json.Unmarshal(data, &ev)
	return ev, err
}
