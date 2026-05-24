package realtime

import "context"

// NoopPublisher discards events (used when Redis pubsub is unavailable in tests).
type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, ProgressEvent) error { return nil }
