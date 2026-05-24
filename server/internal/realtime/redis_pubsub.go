package realtime

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

const DefaultProgressChannel = "videoguard:video:progress"

type RedisPubSub struct {
	client  *redis.Client
	channel string
}

func NewRedisPubSub(addr, password string, db int, channel string) (*RedisPubSub, error) {
	if channel == "" {
		channel = DefaultProgressChannel
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis pubsub ping: %w", err)
	}
	return &RedisPubSub{client: client, channel: channel}, nil
}

func (r *RedisPubSub) Publish(ctx context.Context, event ProgressEvent) error {
	payload, err := event.Marshal()
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, r.channel, payload).Err()
}

func (r *RedisPubSub) Subscribe(ctx context.Context, handler func(ProgressEvent)) error {
	sub := r.client.Subscribe(ctx, r.channel)
	defer func() { _ = sub.Close() }()

	ch := sub.Channel()
	log.Printf("[realtime] subscribed to redis channel %q", r.channel)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			ev, err := ParseProgressEvent([]byte(msg.Payload))
			if err != nil {
				log.Printf("[realtime] invalid progress payload: %v", err)
				continue
			}
			handler(ev)
		}
	}
}

func (r *RedisPubSub) Close() error {
	return r.client.Close()
}

// Ensure RedisPubSub implements ProgressPublisher.
var _ ProgressPublisher = (*RedisPubSub)(nil)
