// Package rabbitmq provides a client for rabbitmq.
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nawafswe/qstorm/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Client is a rabbitmq client.
type Client struct {
	connection       *amqp.Connection
	channel          *amqp.Channel
	confirmationChan chan amqp.Confirmation
	queue            amqp.Queue
}

// NewClient creates a new rabbitmq client.
func NewClient(_ context.Context, brokerURL string, cfg config.RabbitmqConfig) (Client, error) {
	connection, err := amqp.Dial(brokerURL)
	if err != nil {
		return Client{}, fmt.Errorf("failed to dial rabbitmq: %w", err)
	}
	ch, err := connection.Channel()
	if err != nil {
		return Client{}, fmt.Errorf("failed to open a channel: %w", err)
	}
	var confirmationChan chan amqp.Confirmation
	if cfg.Channel.ConfirmMode {
		err = ch.Confirm(false)
		if err != nil {
			return Client{}, fmt.Errorf("failed to enable confirm mode: %w", err)
		}
		confirmationChan = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	}
	q, err := ch.QueueDeclare(
		cfg.Queue.Name,
		cfg.Queue.Durable,
		cfg.Queue.AutoDelete,
		cfg.Queue.Exclusive,
		cfg.Queue.NoWait,
		cfg.Queue.Args)
	if err != nil {
		return Client{}, fmt.Errorf("failed to declare a queue: %w", err)
	}
	if cfg.Exchange.Name != "" {
		err = ch.ExchangeDeclare(
			cfg.Exchange.Name,
			cfg.Exchange.Kind,
			cfg.Exchange.Durable,
			cfg.Exchange.AutoDelete,
			cfg.Exchange.Internal,
			cfg.Exchange.NoWait,
			cfg.Exchange.Args,
		)
		if err != nil {
			return Client{}, fmt.Errorf("failed to declare an exchange: %w", err)
		}
		err = ch.QueueBind(cfg.Queue.Name, cfg.Publisher.RoutingKey, cfg.Exchange.Name, false, nil)
		if err != nil {
			return Client{}, fmt.Errorf("failed to bind queue to exchange: %w", err)
		}
	}
	return Client{
		connection:       connection,
		channel:          ch,
		queue:            q,
		confirmationChan: confirmationChan,
	}, nil
}

// Publish sends a message to the given rabbitmq queue.
func (c Client) Publish(ctx context.Context, queueConfig config.QueueConfig) error {
	ampqpHeaders, err := headers(queueConfig.Attributes)
	if err != nil {
		return fmt.Errorf("failed to create message headers: %w", err)
	}
	// keeping default values for visibility.
	err = c.channel.PublishWithContext(ctx, queueConfig.Rabbitmq.Publisher.Exchange, queueConfig.Rabbitmq.Publisher.RoutingKey,
		queueConfig.Rabbitmq.Publisher.Mandatory,
		false,
		amqp.Publishing{
			Headers:         ampqpHeaders,
			ContentType:     queueConfig.Rabbitmq.Publisher.ContentType,
			ContentEncoding: "",
			DeliveryMode:    queueConfig.Rabbitmq.Publisher.DeliveryMode,
			Priority:        queueConfig.Rabbitmq.Publisher.Priority,
			CorrelationId:   uuid.NewString(),
			ReplyTo:         "",
			Expiration:      "",
			MessageId:       uuid.NewString(),
			Timestamp:       time.Now().UTC(),
			Type:            "",
			UserId:          "",
			AppId:           "",
			Body:            []byte(queueConfig.Payload),
		})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// if the confirmation channel is set (i.e. confirm mode is enabled), wait for confirmation.
	if c.confirmationChan != nil {
		confirmation := <-c.confirmationChan
		if !confirmation.Ack {
			return fmt.Errorf("message not acknowledged by rabbitmq")
		}
	}
	return nil
}

// Close closes the rabbitmq client and its underlying connections.
func (c Client) Close() error {
	return c.connection.Close()
}

// headers convert the given string to amqp.Table.
func headers(attrs string) (amqp.Table, error) {
	if attrs == "" {
		return nil, nil
	}
	var jsonAttrs map[string]any
	err := json.Unmarshal([]byte(attrs), &jsonAttrs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message attributes: %w", err)
	}
	return jsonAttrs, nil
}
