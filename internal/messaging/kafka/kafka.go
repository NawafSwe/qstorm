package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	conkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/google/uuid"
	"github.com/nawafswe/qstorm/internal/config"
)

// librdkafka configuration keys.
// https://github.com/confluentinc/librdkafka/blob/master/CONFIGURATION.md
const (
	bootstrapServers = "bootstrap.servers"
	securityProtocol = "security.protocol"
	saslMechanism    = "sasl.mechanism"
	saslUsername     = "sasl.username"
	saslPassword     = "sasl.password"
	acks             = "acks"
	compressionType  = "compression.type"
	lingerMs         = "linger.ms"
	batchSize        = "batch.size"
)

type Client struct {
	producer *conkafka.Producer
}

// NewClient creates a new kafka client.
func NewClient(_ context.Context, kafkaConnConfig config.KafkaConnectionConfig, kafkaConfig config.KafkaConfig) (Client, error) {
	producerCfg := applyConfig(kafkaConnConfig, kafkaConfig.Producer)
	p, err := conkafka.NewProducer(&producerCfg)
	if err != nil {
		return Client{}, fmt.Errorf("failed to create kafka producer: %w", err)
	}
	return Client{
		producer: p,
	}, nil
}

// Publish sends a message to the given kafka topic.
func (c Client) Publish(_ context.Context, queueConfig config.QueueConfig) error {
	kafkaConfig := queueConfig.Kafka
	msgID := uuid.New().String()
	headers, err := messageHeaders(queueConfig.Attributes)
	if err != nil {
		return fmt.Errorf("failed to create message headers: %w", err)
	}
	// if partition is 0, it means to use any partition.
	partition := int32(kafkaConfig.Partition)
	if partition == 0 {
		partition = conkafka.PartitionAny
	}

	deliveryChan := make(chan conkafka.Event, 1)
	// keeping the zero-values as is for visibility.
	err = c.producer.Produce(&conkafka.Message{
		TopicPartition: conkafka.TopicPartition{
			Topic:     &kafkaConfig.Topic,
			Partition: partition,
			Offset:    0,
			Metadata:  nil,
			Error:     nil,
		},
		Value:         []byte(queueConfig.Payload),
		Key:           []byte(msgID),
		Timestamp:     time.Now().UTC(),
		TimestampType: 0,
		Opaque:        nil,
		Headers:       headers,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("failed to produce message to topic %s: %w", kafkaConfig.Topic, err)
	}
	deliveredEvent := <-deliveryChan
	m, ok := deliveredEvent.(*conkafka.Message)
	if !ok {
		return fmt.Errorf("unexpected delivery event type for topic %s", kafkaConfig.Topic)
	}
	if m.TopicPartition.Error != nil {
		return fmt.Errorf("failed delivering message to topic %s: %w", kafkaConfig.Topic, m.TopicPartition.Error)
	}
	return nil
}

// Close closes the kafka client.
func (c Client) Close() error {
	c.producer.Close()
	return nil
}

func applyConfig(conn config.KafkaConnectionConfig, producer config.KafkaProducerConfig) conkafka.ConfigMap {
	cfg := conkafka.ConfigMap{
		bootstrapServers: conn.BootstrapServers,
		acks:             -1, // default librdkafka
	}

	// connection/auth
	if conn.SecurityProtocol != "" {
		cfg[securityProtocol] = conn.SecurityProtocol
	}
	if conn.SASLMechanism != "" {
		cfg[saslMechanism] = conn.SASLMechanism
	}
	if conn.SASLUsername != "" {
		cfg[saslUsername] = conn.SASLUsername
	}
	if conn.SASLPassword != "" {
		cfg[saslPassword] = conn.SASLPassword.GetValue()
	}

	// producer tuning
	if producer.Acks != nil {
		cfg[acks] = *producer.Acks
	}
	if producer.CompressionType != "" {
		cfg[compressionType] = producer.CompressionType
	}
	if producer.LingerMs != 0 {
		cfg[lingerMs] = producer.LingerMs
	}
	if producer.BatchSize != 0 {
		cfg[batchSize] = producer.BatchSize
	}

	return cfg
}

func messageHeaders(attributes string) ([]conkafka.Header, error) {
	if attributes == "" {
		return nil, nil
	}

	var jsonAttrs map[string]string
	if err := json.Unmarshal([]byte(attributes), &jsonAttrs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message attributes: %w", err)
	}
	headers := make([]conkafka.Header, 0, len(jsonAttrs))

	for k, v := range jsonAttrs {
		headers = append(headers, conkafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}
	return headers, nil
}
