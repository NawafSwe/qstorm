// Package messaging defines interfaces and types for queue message publishing.
package messaging

import (
	"context"

	"github.com/nawafswe/qstorm/internal/config"
)

// Publisher is an interface for publishing messages to a topic.
type Publisher interface {
	// Publish publishes a message to a topic.
	Publish(ctx context.Context, queueConfig config.QueueConfig) error
}

// Closer is an interface for closing resources.
type Closer interface {
	Close() error
}

// Messenger is an interface for publishing messages to a topic.
type Messenger interface {
	Publisher
	Closer
}
