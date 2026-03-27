//go:build integration

package rabbitmq_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/messaging/rabbitmq"
	"github.com/stretchr/testify/assert"
)

const (
	brokerURL = "amqp://guest:guest@localhost:5672/"
	testQueue = "qstorm-integration-test"
)

var (
	testPayload = `{"order_id": "123", "customer_id": "456", "amount": 10}`
	testAttrs   = `{"SOURCE": "qstorm-test", "ENV": "integration"}`
)

func newTestClient(t *testing.T, cfg config.RabbitmqConfig) rabbitmq.Client {
	t.Helper()
	client, err := rabbitmq.NewClient(context.Background(), brokerURL, cfg)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func defaultQueueConfig() config.RabbitmqQueueConfig {
	return config.RabbitmqQueueConfig{Name: testQueue}
}

func TestClient_NewClient(t *testing.T) {
	tests := map[string]struct {
		brokerURL string
		cfg       config.RabbitmqConfig
		wantErr   bool
		errMsg    string
	}{
		"connects with valid credentials": {
			brokerURL: brokerURL,
			cfg: config.RabbitmqConfig{
				Queue: defaultQueueConfig(),
			},
		},
		"connects with confirm mode enabled": {
			brokerURL: brokerURL,
			cfg: config.RabbitmqConfig{
				Queue:   defaultQueueConfig(),
				Channel: config.RabbitmqChannelConfig{ConfirmMode: true},
			},
		},
		"connects and declares durable queue": {
			brokerURL: brokerURL,
			cfg: config.RabbitmqConfig{
				Queue: config.RabbitmqQueueConfig{
					Name:    testQueue + "-durable",
					Durable: true,
				},
			},
		},
		"connects and declares exchange with binding": {
			brokerURL: brokerURL,
			cfg: config.RabbitmqConfig{
				Queue: config.RabbitmqQueueConfig{Name: testQueue + "-exchange"},
				Exchange: config.RabbitmqExchangeConfig{
					Name: "qstorm-test-exchange",
					Kind: "direct",
				},
				Publisher: config.RabbitmqPublisherConfig{
					RoutingKey: testQueue + "-exchange",
				},
			},
		},
		"fails with invalid broker URL": {
			brokerURL: "amqp://guest:guest@localhost:9999/",
			cfg: config.RabbitmqConfig{
				Queue: defaultQueueConfig(),
			},
			wantErr: true,
			errMsg:  "failed to dial rabbitmq",
		},
		"fails with invalid credentials": {
			brokerURL: "amqp://wrong:wrong@localhost:5672/",
			cfg: config.RabbitmqConfig{
				Queue: defaultQueueConfig(),
			},
			wantErr: true,
			errMsg:  "failed to dial rabbitmq",
		},
		"fails with conflicting queue properties": {
			brokerURL: brokerURL,
			cfg: config.RabbitmqConfig{
				Queue: config.RabbitmqQueueConfig{
					Name:    testQueue,
					Durable: true,
					Args:    map[string]any{"x-queue-type": "quorum"},
				},
			},
			wantErr: true,
			errMsg:  "failed to declare a queue",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := rabbitmq.NewClient(context.Background(), tc.brokerURL, tc.cfg)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				assert.NoError(t, err)
				_ = client.Close()
			}
		})
	}
}

func TestClient_Publish(t *testing.T) {
	tests := map[string]struct {
		getClient   func(t *testing.T) rabbitmq.Client
		queueConfig config.QueueConfig
		expectedErr error
	}{
		"publishes with confirm mode enabled": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue:     defaultQueueConfig(),
					Channel:   config.RabbitmqChannelConfig{ConfirmMode: true},
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				})
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				},
			},
		},
		"publishes with confirm mode disabled": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue:     defaultQueueConfig(),
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				})
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				},
			},
		},
		"publishes with empty payload": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue:     defaultQueueConfig(),
					Channel:   config.RabbitmqChannelConfig{ConfirmMode: true},
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				})
			},
			queueConfig: config.QueueConfig{
				Payload: ``,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				},
			},
		},
		"publishes with no attributes": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue:     defaultQueueConfig(),
					Channel:   config.RabbitmqChannelConfig{ConfirmMode: true},
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				})
			},
			queueConfig: config.QueueConfig{
				Payload: testPayload,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				},
			},
		},
		"publishes with content type": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue:     defaultQueueConfig(),
					Channel:   config.RabbitmqChannelConfig{ConfirmMode: true},
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue, ContentType: "application/json"},
				})
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue, ContentType: "application/json"},
				},
			},
		},
		"publishes with persistent delivery mode": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue:     config.RabbitmqQueueConfig{Name: testQueue + "-durable", Durable: true},
					Channel:   config.RabbitmqChannelConfig{ConfirmMode: true},
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue + "-durable", DeliveryMode: 2},
				})
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue + "-durable", DeliveryMode: 2},
				},
			},
		},
		"publishes via exchange with routing key": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue: config.RabbitmqQueueConfig{Name: testQueue + "-via-exchange"},
					Exchange: config.RabbitmqExchangeConfig{
						Name: "qstorm-publish-test-exchange",
						Kind: "direct",
					},
					Channel:   config.RabbitmqChannelConfig{ConfirmMode: true},
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue + "-via-exchange"},
				})
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue + "-via-exchange"},
				},
			},
		},
		"should fail publish with invalid routing key": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue:     defaultQueueConfig(),
					Channel:   config.RabbitmqChannelConfig{ConfirmMode: true},
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				})
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: testAttrs,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: "invalid-routing-key"},
					Exchange:  config.RabbitmqExchangeConfig{Name: "-"},
					Channel:   config.RabbitmqChannelConfig{ConfirmMode: true},
				},
			},
			expectedErr: fmt.Errorf("message not acknowledged by rabbitmq"),
		},
		"fails with invalid attributes json": {
			getClient: func(t *testing.T) rabbitmq.Client {
				return newTestClient(t, config.RabbitmqConfig{
					Queue:     defaultQueueConfig(),
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				})
			},
			queueConfig: config.QueueConfig{
				Payload:    testPayload,
				Attributes: `not-json`,
				Rabbitmq: config.RabbitmqConfig{
					Publisher: config.RabbitmqPublisherConfig{RoutingKey: testQueue},
				},
			},
			expectedErr: fmt.Errorf("failed to create message headers: failed to unmarshal message attributes: invalid character 'o' in literal null (expecting 'u')"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
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
		client, err := rabbitmq.NewClient(context.Background(), brokerURL, config.RabbitmqConfig{
			Queue: defaultQueueConfig(),
		})
		assert.NoError(t, err)
		assert.NoError(t, client.Close())
	})

	t.Run("close twice returns error", func(t *testing.T) {
		client, err := rabbitmq.NewClient(context.Background(), brokerURL, config.RabbitmqConfig{
			Queue: defaultQueueConfig(),
		})
		assert.NoError(t, err)
		assert.NoError(t, client.Close())
		assert.Error(t, client.Close())
	})
}
