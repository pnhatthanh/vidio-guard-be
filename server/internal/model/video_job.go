package model

import "github.com/google/uuid"

// VideoJob is an in-memory job passed to the video processor worker pipeline.
type VideoJob struct {
	VideoID   uuid.UUID
	VideoPath string
	ObjectKey string
}
