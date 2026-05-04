package app

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/pnhatthanh/vidio-guard-be/internal/config"
)

type Worker struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

func NewWorker(redisCfg *config.RedisConfig, asynqCfg *config.AsynqConfig) (*Worker, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisCfg.Addr,
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	}
	return &Worker{
		srv: asynq.NewServer(redisOpt, asynq.Config{
			Concurrency: asynqCfg.Concurrency,
			Queues: map[string]int{
				asynqCfg.Queue: 1,
			},
		}),
		mux: asynq.NewServeMux(),
	}, nil
}

func (w *Worker) RegisterHandler(taskType string, fn func(context.Context, *asynq.Task) error) {
	w.mux.HandleFunc(taskType, fn)
}

func (w *Worker) Start() error {
	if w == nil || w.srv == nil || w.mux == nil {
		return fmt.Errorf("worker not configured")
	}
	return w.srv.Start(w.mux)
}

func (w *Worker) Shutdown() {
	if w == nil || w.srv == nil {
		return
	}
	w.srv.Shutdown()
}
