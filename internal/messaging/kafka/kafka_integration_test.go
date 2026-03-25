//go:build integration

package kafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/messaging/kafka"
	"github.com/stretchr/testify/assert"
)

const (
	plaintextBroker = "localhost:9092"
	saslBroker      = "localhost:9093"
	testTopic       = "qstorm-integration-test"
)

func newPlaintextClient(t *testing.T) kafka.Client {
	t.Helper()
	client, err := kafka.NewClient(context.Background(),
		config.KafkaConnectionConfig{
			BootstrapServers: plaintextBroker,
		},
		config.KafkaConfig{Topic: testTopic},
	)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func newSASLClient(t *testing.T) kafka.Client {
	t.Helper()
	client, err := kafka.NewClient(context.Background(),
		config.KafkaConnectionConfig{
			BootstrapServers: saslBroker,
			SecurityProtocol: "SASL_PLAINTEXT",
			SASLMechanism:    "PLAIN",
			SASLUsername:     "qstorm",
			SASLPassword:     "qstorm-secret",
		},
		config.KafkaConfig{Topic: testTopic},
	)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestClient_Publish(t *testing.T) {
	tests := map[string]struct {
		getClient   func(t *testing.T) kafka.Client
		queueConfig config.QueueConfig
		wantErr     bool
	}{
		"plaintext: publishes message successfully": {
			getClient: newPlaintextClient,
			queueConfig: config.QueueConfig{
				Kafka:      config.KafkaConfig{Topic: testTopic},
				Payload:    `{"order_id":"123"}`,
				Attributes: `{"SOURCE":"qstorm-test"}`,
			},
		},
		"plaintext: publishes with empty payload": {
			getClient: newPlaintextClient,
			queueConfig: config.QueueConfig{
				Kafka:      config.KafkaConfig{Topic: testTopic},
				Payload:    ``,
				Attributes: `{"SOURCE":"qstorm-test"}`,
			},
		},
		"plaintext: publishes with no attributes": {
			getClient: newPlaintextClient,
			queueConfig: config.QueueConfig{
				Kafka:   config.KafkaConfig{Topic: testTopic},
				Payload: `{"order_id":"456"}`,
			},
		},
		"plaintext: publishes to specific partition": {
			getClient: newPlaintextClient,
			queueConfig: config.QueueConfig{
				Kafka:      config.KafkaConfig{Topic: testTopic, Partition: 0},
				Payload:    `{"order_id":"789"}`,
				Attributes: `{"SOURCE":"qstorm-test"}`,
			},
		},
		"plaintext: fails with invalid attributes json": {
			getClient: newPlaintextClient,
			queueConfig: config.QueueConfig{
				Kafka:      config.KafkaConfig{Topic: testTopic},
				Payload:    `{}`,
				Attributes: `not-json`,
			},
			wantErr: true,
		},
		"sasl: publishes with auth": {
			getClient: newSASLClient,
			queueConfig: config.QueueConfig{
				Kafka:      config.KafkaConfig{Topic: testTopic},
				Payload:    `{"order_id":"sasl-123"}`,
				Attributes: `{"SOURCE":"qstorm-sasl-test"}`,
			},
		},
		"sasl: publishes with key": {
			getClient: newSASLClient,
			queueConfig: config.QueueConfig{
				Kafka:      config.KafkaConfig{Topic: testTopic, Key: "order-key"},
				Payload:    `{"order_id":"sasl-456"}`,
				Attributes: `{"SOURCE":"qstorm-sasl-test","ENV":"integration"}`,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := tc.getClient(t)
			err := client.Publish(context.Background(), tc.queueConfig)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKafkaClient_Publish_InvalidCredentials(t *testing.T) {
	// librdkafka handles SASL auth failures asynchronously — Produce() enqueues
	// the message but the delivery never arrives.
	// With context cancellation in Publish, it returns context.DeadlineExceeded.
	client, err := kafka.NewClient(context.Background(),
		config.KafkaConnectionConfig{
			BootstrapServers: saslBroker,
			SecurityProtocol: "SASL_PLAINTEXT",
			SASLMechanism:    "PLAIN",
			SASLUsername:     "wrong-user",
			SASLPassword:     "wrong-password",
		},
		config.KafkaConfig{Topic: testTopic},
	)
	assert.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = client.Publish(ctx, config.QueueConfig{
		Kafka:      config.KafkaConfig{Topic: testTopic},
		Payload:    `{"should":"fail"}`,
		Attributes: `{"SOURCE":"test"}`,
	})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
