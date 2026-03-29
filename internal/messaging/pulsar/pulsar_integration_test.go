//go:build integration

package pulsar_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/messaging/pulsar"
	"github.com/stretchr/testify/assert"
)

const (
	brokerURL = "pulsar://localhost:6650"
	testTopic = "qstorm-integration-test"
)

var (
	testPayload = `{"order_id": "123", "customer_id": "456", "amount": 10}`
	testAttrs   = `{"SOURCE": "qstorm-test", "ENV": "integration"}`
)

func defaultPulsarConfig() config.PulsarConfig {
	return config.PulsarConfig{Topic: testTopic, OperationTimeout: time.Second * 3}
}

func defaultConnConfig() config.PulsarConnectionConfig {
	return config.PulsarConnectionConfig{URL: brokerURL}
}

func newTestClient(t *testing.T, cfg config.PulsarConfig) pulsar.Client {
	t.Helper()
	client, err := pulsar.NewClient(context.Background(), cfg, defaultConnConfig())
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestClient_NewClient(t *testing.T) {
	tests := map[string]struct {
		cfg         config.PulsarConfig
		conn        config.PulsarConnectionConfig
		expectedErr error
	}{
		"connects with valid config": {
			cfg:  defaultPulsarConfig(),
			conn: defaultConnConfig(),
		},
		"connects with producer name": {
			cfg: config.PulsarConfig{
				Topic: testTopic,
				Publisher: config.PulsarPublisherConfig{
					Name: "qstorm-test-producer",
				},
			},
			conn: defaultConnConfig(),
		},
		"connects with batching disabled": {
			cfg: config.PulsarConfig{
				Topic: testTopic,
				Publisher: config.PulsarPublisherConfig{
					DisableBatching: true,
				},
			},
			conn: defaultConnConfig(),
		},
		"fails with invalid broker URL": {
			cfg:         defaultPulsarConfig(),
			conn:        config.PulsarConnectionConfig{URL: "pulsar://localhost:9999"},
			expectedErr: fmt.Errorf("failed to create pulsar producer"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client, err := pulsar.NewClient(context.Background(), tc.cfg, tc.conn)
			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				_ = client.Close()
			}
		})
	}
}

func TestClient_Publish(t *testing.T) {
	tests := map[string]struct {
		getClient   func(t *testing.T) pulsar.Client
		queueConfig config.QueueConfig
		expectedErr error
	}{
		"publishes message successfully": {
			getClient: func(t *testing.T) pulsar.Client {
				return newTestClient(t, defaultPulsarConfig())
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Pulsar:     defaultPulsarConfig(),
			},
		},
		"publishes with partition key": {
			getClient: func(t *testing.T) pulsar.Client {
				return newTestClient(t, defaultPulsarConfig())
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Pulsar: config.PulsarConfig{
					Topic:        testTopic,
					PartitionKey: "customer-123",
				},
			},
		},
		"publishes with ordering key": {
			getClient: func(t *testing.T) pulsar.Client {
				return newTestClient(t, defaultPulsarConfig())
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Pulsar: config.PulsarConfig{
					Topic: testTopic,
					Publisher: config.PulsarPublisherConfig{
						OrderingKey: "order-group",
					},
				},
			},
		},
		"publishes with empty payload": {
			getClient: func(t *testing.T) pulsar.Client {
				return newTestClient(t, defaultPulsarConfig())
			},
			queueConfig: config.QueueConfig{
				Payload: ``,
				Pulsar:  defaultPulsarConfig(),
			},
		},
		"publishes with no attributes": {
			getClient: func(t *testing.T) pulsar.Client {
				return newTestClient(t, defaultPulsarConfig())
			},
			queueConfig: config.QueueConfig{
				Payload: testPayload,
				Pulsar:  defaultPulsarConfig(),
			},
		},
		"fails with invalid attributes json": {
			getClient: func(t *testing.T) pulsar.Client {
				return newTestClient(t, defaultPulsarConfig())
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: `not-json`,
				Pulsar:     defaultPulsarConfig(),
			},
			expectedErr: fmt.Errorf("failed to create message properties: failed to unmarshal message properties: invalid character 'o' in literal null (expecting 'u')"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := tc.getClient(t)
			err := client.Publish(context.Background(), tc.queueConfig)

			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	t.Run("closes without error", func(t *testing.T) {
		t.Parallel()
		client, err := pulsar.NewClient(context.Background(), defaultPulsarConfig(), defaultConnConfig())
		assert.NoError(t, err)
		assert.NoError(t, client.Close())
	})
}
