package kafka

import (
	"fmt"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/nawafswe/qstorm/internal/config"
	"github.com/stretchr/testify/assert"
)

func intPtr(v int) *int { return &v }

func Test_applyConfig(t *testing.T) {
	tests := map[string]struct {
		conn     config.KafkaConnectionConfig
		producer config.KafkaProducerConfig
		want     kafka.ConfigMap
	}{
		"bootstrap servers only with default acks": {
			conn: config.KafkaConnectionConfig{
				BootstrapServers: "localhost:9092",
			},
			want: kafka.ConfigMap{
				"bootstrap.servers": "localhost:9092",
				"acks":              -1,
				"log_level":         0,
			},
		},
		"full SASL_SSL connection": {
			conn: config.KafkaConnectionConfig{
				BootstrapServers: "broker.cloud:9093",
				SecurityProtocol: "SASL_SSL",
				SASLMechanism:    "PLAIN",
				SASLUsername:     "api-key",
				SASLPassword:     config.NonLoggable("api-secret"),
			},
			want: kafka.ConfigMap{
				"bootstrap.servers": "broker.cloud:9093",
				"security.protocol": "SASL_SSL",
				"sasl.mechanism":    "PLAIN",
				"sasl.username":     "api-key",
				"sasl.password":     "api-secret",
				"acks":              -1,
				"log_level":         0,
			},
		},
		"producer tuning options": {
			conn: config.KafkaConnectionConfig{
				BootstrapServers: "localhost:9092",
			},
			producer: config.KafkaProducerConfig{
				Acks:            intPtr(1),
				CompressionType: "snappy",
				LingerMs:        10,
				BatchSize:       32768,
			},
			want: kafka.ConfigMap{
				"bootstrap.servers": "localhost:9092",
				"acks":              1,
				"compression.type":  "snappy",
				"linger.ms":         10,
				"batch.size":        32768,
				"log_level":         0,
			},
		},
		"acks zero for fire-and-forget": {
			conn: config.KafkaConnectionConfig{
				BootstrapServers: "localhost:9092",
			},
			producer: config.KafkaProducerConfig{
				Acks: intPtr(0),
			},
			want: kafka.ConfigMap{
				"bootstrap.servers": "localhost:9092",
				"acks":              0,
				"log_level":         0,
			},
		},
		"nil acks keeps default": {
			conn: config.KafkaConnectionConfig{
				BootstrapServers: "localhost:9092",
			},
			producer: config.KafkaProducerConfig{
				Acks: nil,
			},
			want: kafka.ConfigMap{
				"bootstrap.servers": "localhost:9092",
				"acks":              -1,
				"log_level":         0,
			},
		},
		"zero producer values are omitted": {
			conn: config.KafkaConnectionConfig{
				BootstrapServers: "localhost:9092",
			},
			producer: config.KafkaProducerConfig{
				CompressionType: "",
				LingerMs:        0,
				BatchSize:       0,
			},
			want: kafka.ConfigMap{
				"bootstrap.servers": "localhost:9092",
				"acks":              -1,
				"log_level":         0,
			},
		},
		"partial connection config": {
			conn: config.KafkaConnectionConfig{
				BootstrapServers: "localhost:9092",
				SecurityProtocol: "SASL_SSL",
			},
			want: kafka.ConfigMap{
				"bootstrap.servers": "localhost:9092",
				"security.protocol": "SASL_SSL",
				"acks":              -1,
				"log_level":         0,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := applyConfig(tc.conn, tc.producer)
			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_messageHeaders(t *testing.T) {
	tests := map[string]struct {
		attrs       string
		want        []kafka.Header
		expectedErr error
	}{
		"should successfully parse message attributes": {
			attrs: `{"SOURCE":"qstorm-test"}`,
			want: []kafka.Header{
				{
					Key:   "SOURCE",
					Value: []byte("qstorm-test"),
				},
			},
		},
		"should return empty headers when no attributes passed": {
			attrs: ``,
		},
		"should fail parsing message attributes when invalid json string passed": {
			attrs:       `{"SOURCE":"qstorm-test#`,
			expectedErr: fmt.Errorf("failed to unmarshal message attributes: unexpected end of JSON input"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := messageHeaders(tc.attrs)
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
