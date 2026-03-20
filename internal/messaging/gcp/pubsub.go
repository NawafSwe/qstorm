package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub/v2"
	"github.com/nawafswe/qstorm/internal/messaging"
	"google.golang.org/api/option"
	pubsub2 "google.golang.org/genproto/googleapis/pubsub/v1"
)

type Option func(*Client)

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

type Client struct {
	client *pubsub.Client
	opts   options
}

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

func (c Client) Publish(ctx context.Context, topic string, message messaging.Message) error {
	publisher := c.client.Publisher(topic)

	result := publisher.Publish(ctx, &pubsub.Message{
		ID:          message.ID,
		Data:        message.Data,
		Attributes:  message.Attributes,
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
