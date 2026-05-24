package dto

import "github.com/google/uuid"

type VideoJob struct {
	VideoID   uuid.UUID
	VideoPath string
	ObjectKey string
}
