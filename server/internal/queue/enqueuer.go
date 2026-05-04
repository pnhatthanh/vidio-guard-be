package queue

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/pnhatthanh/vidio-guard-be/internal/worker"
)

type Enqueuer interface {
	EnqueueVideoProcess(videoID, objectKey string) error
}

type asynqEnqueuer struct {
	client      *asynq.Client
	queue       string
	maxRetry    int
	taskTimeout time.Duration
}

func NewAsynqEnqueuer(client *asynq.Client, queue string, maxRetry int, taskTimeout time.Duration) *asynqEnqueuer {
	return &asynqEnqueuer{
		client:      client,
		queue:       queue,
		maxRetry:    maxRetry,
		taskTimeout: taskTimeout,
	}
}

func (e *asynqEnqueuer) EnqueueVideoProcess(videoID, objectKey string) error {
	if videoID == "" {
		return fmt.Errorf("videoID is required")
	}
	if objectKey == "" {
		return fmt.Errorf("objectKey is required")
	}

	t, err := worker.NewVideoProcessTask(videoID, objectKey)
	if err != nil {
		return err
	}

	_, err = e.client.Enqueue(
		t,
		asynq.Queue(e.queue),
		asynq.MaxRetry(e.maxRetry),
		asynq.Timeout(e.taskTimeout),
	)
	if err != nil {
		return err
	}
	return nil
}
