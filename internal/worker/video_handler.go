package worker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/hibiken/asynq"
	"github.com/pnhatthanh/vidio-guard-be/internal/model"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/services"
)

type VideoProcessHandler struct {
	Processor services.VideoProcessor
	Store     pkg.StoreProvider
	TempDir   string
}

func (h *VideoProcessHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var p VideoProcessPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	ext := filepath.Ext(p.ObjectKey)
	tmpFile, err := os.CreateTemp(h.TempDir, "video-"+p.VideoID+"-")
	if err != nil {
		h.updateFailedStatus(ctx, p.VideoID)
		return err
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()

	if ext != "" {
		newPath := tmpPath + ext
		_ = os.Rename(tmpPath, newPath)
		tmpPath = newPath
	}
	defer os.Remove(tmpPath)

	if err := h.Store.DownloadToFile(ctx, p.ObjectKey, tmpPath); err != nil {
		h.updateFailedStatus(ctx, p.VideoID)
		return err
	}

	job := model.VideoJob{
		VideoID:   p.VideoID,
		VideoPath: tmpPath,
	}
	if err := h.Processor.Process(job); err != nil {
		h.updateFailedStatus(ctx, p.VideoID)
		return err
	}
	return nil
}

func (h *VideoProcessHandler) updateFailedStatus(ctx context.Context, videoID string) {
	retryCount, ok1 := asynq.GetRetryCount(ctx)
	maxRetry, ok2 := asynq.GetMaxRetry(ctx)
	if ok1 && ok2 && retryCount >= maxRetry {
		return
	}
}
