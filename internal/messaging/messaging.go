package messaging

import "context"

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

// Connector is an interface for connecting to a topic.
type Connector interface {
	Connect(ctx context.Context, topic string) error
}

// Messenger is an interface for publishing messages to a topic.
type Messenger interface {
	Publisher
	Connector
	Closer
}
