package worker

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
)

type VideoProcessHandler struct {
	Processing services.VideoProcessingService
}

func (h *VideoProcessHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var p VideoProcessPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	videoID, err := uuid.Parse(p.VideoID)
	if err != nil {
		return err
	}

	err = h.Processing.Process(ctx, videoID, p.ObjectKey)
	if err != nil {
		retryCount, ok1 := asynq.GetRetryCount(ctx)
		maxRetry, ok2 := asynq.GetMaxRetry(ctx)
		if ok1 && ok2 && retryCount < maxRetry {
			return err
		}
		return err
	}
	return nil
}
