package dto

import "github.com/pnhatthanh/vidio-guard-be/internal/constants"

// UploadVideoResponse is returned after a successful video upload.
type UploadVideoResponse struct {
	VideoID string                 `json:"video_id"`
	Status  constants.VideoStatus  `json:"status"`
}
