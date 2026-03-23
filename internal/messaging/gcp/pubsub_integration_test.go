//go:build integration

package gcp_test

import (
	"context"
	"os"
	"testing"

	"github.com/nawafswe/qstorm/internal/messaging"
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

func newTestClient(t *testing.T) gcp.Client {
	t.Helper()
	setupEmulator(t)
	client, err := gcp.NewClient(context.Background(), testProjectID)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestClient_NewClient(t *testing.T) {
	tests := map[string]struct {
		projectID   string
		opts        []gcp.Option
		setup       func(t *testing.T)
		expectedErr bool
	}{
		"creates client with emulator": {
			projectID: testProjectID,
			setup:     setupEmulator,
		},

		"creates client with service account credentials": {
			projectID: testProjectID,
			opts: func() []gcp.Option {
				creds := `{"type":"service_account","project_id":"qstorm-project"}`
				return []gcp.Option{gcp.WithServiceAccountCredentials(&creds)}
			}(),
			setup: setupEmulator,
		},

		"creates client with nil credentials": {
			projectID: testProjectID,
			opts:      []gcp.Option{gcp.WithServiceAccountCredentials(nil)},
			setup:     setupEmulator,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			client, err := gcp.NewClient(context.Background(), tc.projectID, tc.opts...)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				_ = client.Close()
			}
		})
	}
}

func TestClient_Publish(t *testing.T) {
	tests := map[string]struct {
		topic   string
		message messaging.Message
		wantErr bool
	}{
		"publishes message successfully": {
			topic: testTopic,
			message: messaging.Message{
				ID:         "test-id-1",
				Data:       []byte(`{"order_id":"123"}`),
				Attributes: `{"SOURCE":"qstorm-test"}`,
			},
		},

		"fails with ordering key when not enabled": {
			topic: testTopic,
			message: messaging.Message{
				ID:          "test-id-2",
				Data:        []byte(`{"order_id":"456"}`),
				Attributes:  `{"SOURCE":"qstorm-test"}`,
				OrderingKey: "order-key",
			},
			wantErr: true,
		},

		"publishes with empty data": {
			topic: testTopic,
			message: messaging.Message{
				ID:         "test-id-3",
				Data:       []byte(``),
				Attributes: `{"SOURCE":"qstorm-test"}`,
			},
		},

		"fails with invalid attributes json": {
			topic: testTopic,
			message: messaging.Message{
				ID:         "test-id-4",
				Data:       []byte(`{}`),
				Attributes: `not-json`,
			},
			wantErr: true,
		},

		"fails with non-existent topic": {
			topic: "projects/qstorm-project/topics/non-existent",
			message: messaging.Message{
				ID:         "test-id-5",
				Data:       []byte(`{}`),
				Attributes: `{"SOURCE":"test"}`,
			},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := newTestClient(t)
			err := client.Publish(context.Background(), tc.topic, tc.message)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	t.Run("closes without error", func(t *testing.T) {
		client := newTestClient(t)
		err := client.Close()
		assert.NoError(t, err)
	})

	t.Run("close twice returns error", func(t *testing.T) {
		setupEmulator(t)
		client, err := gcp.NewClient(context.Background(), testProjectID)
		assert.NoError(t, err)

		assert.NoError(t, client.Close())
		assert.Error(t, client.Close())
	})
}
