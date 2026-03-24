// Package messaging defines interfaces and types for queue message publishing.
package messaging

// Message is a message to be published by a Publisher.
type Message struct {
	ID          string
	Data        []byte
	Attributes  string
	OrderingKey string
}

// Publisher is an interface for publishing messages to a topic.
type Publisher interface {
	// Publish publishes a message to a topic.
	Publish(ctx context.Context, topic string, message Message) error
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
