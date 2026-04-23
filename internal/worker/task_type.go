package worker

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const TypeVideoProcess = "video:process"

type VideoProcessPayload struct {
	VideoID   string `json:"video_id"`
	ObjectKey string `json:"object_key"`
}

func NewVideoProcessTask(videoID, objectKey string) (*asynq.Task, error) {
	if videoID == "" {
		return nil, fmt.Errorf("videoID is required")
	}
	if objectKey == "" {
		return nil, fmt.Errorf("objectKey is required")
	}
	payload, err := json.Marshal(VideoProcessPayload{
		VideoID: videoID, 
		ObjectKey: objectKey,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeVideoProcess, payload), nil
}
