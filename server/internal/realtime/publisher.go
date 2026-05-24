package realtime

import "context"

type ProgressPublisher interface {
	Publish(ctx context.Context, event ProgressEvent) error
}
