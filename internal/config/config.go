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
	GCPPubSub QueueType = "gcp-pubsub"
)

// Config is the configuration for the application.
type Config struct {
	Queue      QueueConfig      `mapstructure:"QUEUE"`
	Stages     []StageConfig    `mapstructure:"STAGES"`
	Connection ConnectionConfig `mapstructure:"CONNECTION"`
}

// QueueConfig configuration for the queue run.
type QueueConfig struct {
	Topic       string    `mapstructure:"TOPIC"`
	OrderingKey string    `mapstructure:"ORDERING_KEY"`
	Type        QueueType `mapstructure:"TYPE"`
	Payload     string    `mapstructure:"PAYLOAD"`
	Attributes  string    `mapstructure:"ATTRIBUTES"`
}

// ConnectionConfig holds connection details.
// Different queue types use different fields:
//   - gcp-pubsub: ProjectID (required), CredentialsFile (optional), EmulatorHost (optional)
//   - kafka/rabbitmq: Brokers (future)
type ConnectionConfig struct {
	PubSub PubSubConfig `mapstructure:"PUBSUB"`
}

type PubSubConfig struct {
	ProjectID       string      `mapstructure:"PROJECT_ID" json:",omitempty"`
	CredentialsFile NonLoggable `mapstructure:"CREDENTIALS_FILE" json:",omitempty"`
	EmulatorHost    string      `mapstructure:"EMULATOR_HOST" json:",omitempty"`
}

// StageConfig configuration for a stage run.
type StageConfig struct {
	Duration        time.Duration `mapstructure:"DURATION"`
	StartAtDuration time.Duration `mapstructure:"START_AT_DURATION"`
	Rate            int           `mapstructure:"RATE"`
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

// LoadConnConfigFromEnv loads connection credentials from an env file.
func LoadConnConfigFromEnv(path string) (ConnectionConfig, error) {

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
