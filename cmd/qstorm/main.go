package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nawafswe/qstorm/internal/messaging"
	"github.com/nawafswe/qstorm/internal/messaging/gcp"
)

func main() {
	fmt.Println("==== starting qstorm")

	pubsubClient, err := gcp.NewClient(context.Background(), "qstorm-project")
	if err != nil {
		fmt.Println("failed to create pubsub client: ", err)
		return
	}
	err = pubsubClient.Connect(context.Background(), "qstorm-topic")
	if err != nil {
		fmt.Println("failed to connect to pubsub topic: ", err)
		return
	}
	fmt.Println("==== qstorm started")
	err = pubsubClient.Publish(context.Background(), "qstorm-topic", messaging.Message{
		ID:   uuid.NewString(),
		Data: []byte(`{}`),
	})
	if err != nil {
		fmt.Println("failed to publish message: ", err)
		return
	}
}
