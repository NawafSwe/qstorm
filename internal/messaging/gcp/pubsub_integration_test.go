//go:build integration

package gcp_test

import (
	"context"
	"os"
	"testing"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/messaging/gcp"
	"github.com/stretchr/testify/assert"
)

const (
	testEmulatorHost = "localhost:8095"
	testProjectID    = "qstorm-project"
	testTopic        = "projects/qstorm-project/topics/qstorm-topic"
)

func setupEmulator(t *testing.T) {
	t.Helper()
	err := os.Setenv("PUBSUB_EMULATOR_HOST", testEmulatorHost)
	assert.NoError(t, err)
}

func newEmulatorClient(t *testing.T) gcp.Client {
	t.Helper()
	setupEmulator(t)
	client, err := gcp.NewClient(context.Background(), testProjectID)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestClient_Publish(t *testing.T) {
	tests := map[string]struct {
		getClient   func(t *testing.T) gcp.Client
		queueConfig config.QueueConfig
		wantErr     bool
	}{
		"publishes message successfully": {
			getClient: newEmulatorClient,
			queueConfig: config.QueueConfig{
				PubSub:     config.PubSubConfig{Topic: testTopic},
				Payload:    `{"order_id":"123"}`,
				Attributes: `{"SOURCE":"qstorm-test"}`,
			},
		},
		"publishes with ordering key": {
			getClient: newEmulatorClient,
			queueConfig: config.QueueConfig{
				PubSub:     config.PubSubConfig{Topic: testTopic, OrderingKey: "order-key"},
				Payload:    `{"order_id":"456"}`,
				Attributes: `{"SOURCE":"qstorm-test"}`,
			},
		},
		"publishes with empty payload": {
			getClient: newEmulatorClient,
			queueConfig: config.QueueConfig{
				PubSub:     config.PubSubConfig{Topic: testTopic},
				Payload:    ``,
				Attributes: `{"SOURCE":"qstorm-test"}`,
			},
		},
		"fails with invalid attributes json": {
			getClient: newEmulatorClient,
			queueConfig: config.QueueConfig{
				PubSub:     config.PubSubConfig{Topic: testTopic},
				Payload:    `{}`,
				Attributes: `not-json`,
			},
			wantErr: true,
		},
		"fails with non-existent topic": {
			getClient: newEmulatorClient,
			queueConfig: config.QueueConfig{
				PubSub:     config.PubSubConfig{Topic: "projects/qstorm-project/topics/non-existent"},
				Payload:    `{}`,
				Attributes: `{"SOURCE":"test"}`,
			},
			wantErr: true,
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
