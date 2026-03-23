// Package gcp provides a Google Cloud PubSub messaging client.
package gcp

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	"github.com/nawafswe/qstorm/internal/messaging"
	"google.golang.org/api/option"
	pubsub2 "google.golang.org/genproto/googleapis/pubsub/v1"
)

// Option configures optional Client behavior.
type Option func(*Client)

// WithServiceAccountCredentials sets the GCP service account JSON credentials for authentication.
func WithServiceAccountCredentials(credentials *string) Option {
	return func(c *Client) {
		if credentials != nil {
			c.opts.ServiceAccountCredentials = credentials
		}
	}
}

type options struct {
	ServiceAccountCredentials *string
}

// Client wraps a Google Cloud PubSub client.
type Client struct {
	client *pubsub.Client
	opts   options
}

// NewClient creates a new PubSub client for the given project.
func NewClient(ctx context.Context, projectID string, opts ...Option) (Client, error) {
	c := Client{}
	for _, opt := range opts {
		opt(&c)
	}
	pubsubClient, err := pubsub.NewClient(ctx, projectID, buildOptions(c.opts)...)
	if err != nil {
		return Client{}, fmt.Errorf("failed to create pubsub client: %w", err)
	}
	c.client = pubsubClient
	return c, nil
}

// Connect verifies that the given topic exists.
func (c Client) Connect(ctx context.Context, topic string) error {
	tt, err := c.client.TopicAdminClient.GetTopic(ctx, &pubsub2.GetTopicRequest{
		Topic: topic,
	})
	if err != nil {
		return fmt.Errorf("failed to get topic: %w", err)
	}
	if tt.GetName() == "" {
		return fmt.Errorf("topic %s does not exist", topic)
	}
	return nil
}

// Publish sends a message to the given PubSub topic and waits for confirmation.
func (c Client) Publish(ctx context.Context, topic string, message messaging.Message) error {
	publisher := c.client.Publisher(topic)

	var attrs map[string]string
	if err := json.Unmarshal([]byte(message.Attributes), &attrs); err != nil {
		return fmt.Errorf("failed to unmarshal attributes: %w", err)
	}
	result := publisher.Publish(ctx, &pubsub.Message{
		ID:          message.ID,
		Data:        message.Data,
		Attributes:  attrs,
		OrderingKey: message.OrderingKey,
	})
	<-result.Ready()
	// ignoring serverID as it is not used.
	_, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed publishing message to topic %s: %w", topic, err)
	}
	return nil
}

// Close shuts down the PubSub client.
func (c Client) Close() error {
	return c.client.Close()
}

// buildOptions function checks set options and return a slice of option.ClientOption.
func buildOptions(opts options) []option.ClientOption {
	var clientOptions []option.ClientOption
	if opts.ServiceAccountCredentials != nil {
		clientOptions = append(clientOptions, option.WithAuthCredentialsJSON(option.ServiceAccount, []byte(*opts.ServiceAccountCredentials)))
	}
	return clientOptions
}
