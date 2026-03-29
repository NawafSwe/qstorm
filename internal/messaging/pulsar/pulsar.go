// Package pulsar provides a pulsar client.
package pulsar

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apacheplsr "github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/nawafswe/qstorm/internal/config"
)

const (
	defaultOperationTimeout = 30 * time.Second
)

type Client struct {
	producer     apacheplsr.Producer
	pulsarClient apacheplsr.Client
}

// NewClient creates a new pulsar client.
func NewClient(_ context.Context, cfg config.PulsarConfig, conn config.PulsarConnectionConfig) (Client, error) {
	authStrategy, err := auth(conn)
	var lgr log.Logger
	if !cfg.EnablePulsarLogging {
		lgr = log.DefaultNopLogger()
	}
	if err != nil {
		return Client{}, fmt.Errorf("failed to create pulsar authentication strategy: %w", err)
	}
	operationTimeout := defaultOperationTimeout
	if cfg.OperationTimeout != 0 {
		operationTimeout = cfg.OperationTimeout
	}
	c, err := apacheplsr.NewClient(apacheplsr.ClientOptions{
		URL:              conn.URL,
		OperationTimeout: operationTimeout,
		Authentication:   authStrategy,
		Logger:           lgr,
	})
	if err != nil {
		return Client{}, fmt.Errorf("failed to create pulsar client: %w", err)
	}
	p, err := c.CreateProducer(apacheplsr.ProducerOptions{
		Topic:                   cfg.Topic,
		Name:                    cfg.Publisher.Name,
		DisableBlockIfQueueFull: cfg.Publisher.DisableBlockIfQueueFull,
		DisableBatching:         cfg.Publisher.DisableBatching,
		BatchingMaxPublishDelay: cfg.Publisher.BatchingMaxPublishDelay,
		BatchingMaxMessages:     cfg.Publisher.BatchingMaxMessages,
		BatchingMaxSize:         cfg.Publisher.BatchingMaxSize,
	})
	if err != nil {
		// close client connection.
		c.Close()
		return Client{}, fmt.Errorf("failed to create pulsar producer for topic %s: %w", cfg.Topic, err)
	}
	return Client{
		producer:     p,
		pulsarClient: c,
	}, nil
}

// Publish sends a message to the given pulsar topic.
func (c Client) Publish(ctx context.Context, queueConfig config.QueueConfig) error {
	msgProps, err := messageProperties(queueConfig.Attributes)
	if err != nil {
		return fmt.Errorf("failed to create message properties: %w", err)
	}
	// keeping the zero-values as is for visibility.
	_, err = c.producer.Send(ctx, &apacheplsr.ProducerMessage{
		Payload:             []byte(queueConfig.Payload),
		Value:               nil,
		Key:                 queueConfig.Pulsar.PartitionKey,
		OrderingKey:         queueConfig.Pulsar.Publisher.OrderingKey,
		Properties:          msgProps,
		EventTime:           time.Now().UTC(),
		ReplicationClusters: nil,
		DisableReplication:  false,
		SequenceID:          nil,
		DeliverAfter:        queueConfig.Pulsar.Publisher.DeliverAfter,
		DeliverAt:           time.Time{},
		Schema:              nil,
		Transaction:         nil,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to topic %s: %w", queueConfig.Pulsar.Topic, err)
	}
	return nil
}

// Close closes the pulsar client.
func (c Client) Close() error {
	c.pulsarClient.Close()
	return nil
}

func messageProperties(attrs string) (map[string]string, error) {
	if attrs == "" {
		return nil, nil
	}
	var jsonAttrs map[string]string
	if err := json.Unmarshal([]byte(attrs), &jsonAttrs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message properties: %w", err)
	}
	return jsonAttrs, nil
}

// auth returns the authentication strategy.
func auth(cfg config.PulsarConnectionConfig) (apacheplsr.Authentication, error) {
	if cfg.AuthToken != nil {
		return apacheplsr.NewAuthenticationToken(*cfg.AuthToken), nil
	}
	if basicAuth := cfg.BasicAuth; basicAuth != nil {
		return apacheplsr.NewAuthenticationBasic(basicAuth.Username, string(basicAuth.Password))
	}
	return nil, nil
}
