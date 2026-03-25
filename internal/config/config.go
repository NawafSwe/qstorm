// Package config provides configuration types and loaders for qstorm.
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// NonLoggable is a string that redacts itself when printed or marshalled to JSON.
type NonLoggable string

func (nl NonLoggable) String() string               { return "redacted" }
func (nl NonLoggable) GoString() string             { return "redacted" }
func (nl NonLoggable) MarshalJSON() ([]byte, error) { return []byte(`"redacted"`), nil }
func (nl NonLoggable) GetValue() string             { return string(nl) }

// QueueType is the type of queue to determine the implementation of the Publisher.
type QueueType string

const (
	GCPPubSub   QueueType = "gcp-pubsub"
	ApacheKafka QueueType = "apache-kafka"
)

// Config is the configuration for the application.
type Config struct {
	Queue      QueueConfig      `mapstructure:"QUEUE"`
	Stages     []StageConfig    `mapstructure:"STAGES"`
	Connection ConnectionConfig `mapstructure:"CONNECTION"`
}

// QueueConfig configuration for the queue run.
type QueueConfig struct {
	Type       QueueType `mapstructure:"TYPE"`
	Payload    string    `mapstructure:"PAYLOAD"`
	Attributes string    `mapstructure:"ATTRIBUTES"`

	PubSub PubSubConfig `mapstructure:"PUBSUB"`
	Kafka  KafkaConfig  `mapstructure:"KAFKA"`
}

// ConnectionConfig holds connection details.
// Different queue types use different fields:
//   - gcp-pubsub: ProjectID (required), CredentialsFile (optional), EmulatorHost (optional)
//   - kafka/rabbitmq: Brokers (future)
type ConnectionConfig struct {
	PubSub PubSubConnectionConfig `mapstructure:"PUBSUB"`
	Kafka  KafkaConnectionConfig  `mapstructure:"KAFKA"`
}

// PubSubConnectionConfig holds Google Cloud PubSub connection details.
type PubSubConnectionConfig struct {
	ProjectID       string      `mapstructure:"PROJECT_ID" json:",omitempty"`
	CredentialsFile NonLoggable `mapstructure:"CREDENTIALS_FILE" json:",omitempty"`
	EmulatorHost    string      `mapstructure:"EMULATOR_HOST" json:",omitempty"`
}

// PubSubConfig holds Google Cloud PubSub configuration.
type PubSubConfig struct {
	Topic       string `mapstructure:"TOPIC"`
	OrderingKey string `mapstructure:"ORDERING_KEY"`
}

// KafkaConnectionConfig holds Kafka connection details.
type KafkaConnectionConfig struct {
	BootstrapServers string      `mapstructure:"BOOTSTRAP_SERVERS"`
	SecurityProtocol string      `mapstructure:"SECURITY_PROTOCOL"`
	SASLMechanism    string      `mapstructure:"SASL_MECHANISM"`
	SASLUsername     string      `mapstructure:"SASL_USERNAME"`
	SASLPassword     NonLoggable `mapstructure:"SASL_PASSWORD"`
}

// KafkaConfig holds Kafka configuration.
type KafkaConfig struct {
	Topic     string              `mapstructure:"TOPIC"`
	Key       string              `mapstructure:"KEY"`
	Partition int                 `mapstructure:"PARTITION"`
	Producer  KafkaProducerConfig `mapstructure:"PRODUCER"`
}

// KafkaProducerConfig holds kafka producer configurations.
type KafkaProducerConfig struct {
	Acks            *int   `mapstructure:"ACKS"`
	CompressionType string `mapstructure:"COMPRESSION_TYPE"`
	LingerMs        int    `mapstructure:"LINGER_MS"`
	BatchSize       int    `mapstructure:"BATCH_SIZE"`
}

// StageConfig configuration for a stage run.
type StageConfig struct {
	Duration time.Duration `mapstructure:"DURATION"`
	Rate     int           `mapstructure:"RATE"`
}

// LoadJSONConfig loads the configuration from the given path.
func LoadJSONConfig(path string) (Config, error) {
	const delimiter = "__"
	vpr := viper.NewWithOptions(viper.KeyDelimiter(delimiter))

	vpr.SetConfigFile(path)
	vpr.SetConfigType("json")

	var cfg Config
	if err := vpr.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("failed reading config file: %w", err)
	}
	if err := vpr.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("failed unmarshalling config: %w", err)
	}

	return cfg, nil
}

// LoadConnConfig loads connection credentials from an env file.
func LoadConnConfig(path string) (ConnectionConfig, error) {
	const delimiter = "__"
	vpr := viper.NewWithOptions(viper.KeyDelimiter(delimiter))
	vpr.SetConfigFile(path)
	vpr.SetConfigType("env")

	var conn ConnectionConfig
	if err := vpr.ReadInConfig(); err != nil {
		return conn, fmt.Errorf("failed reading env file: %w", err)
	}
	if err := vpr.Unmarshal(&conn); err != nil {
		return conn, fmt.Errorf("failed unmarshalling env: %w", err)
	}

	return conn, nil
}
