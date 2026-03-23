package main

import (
	"context"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/printer"
)

type queueStorm func(ctx context.Context, cfg config.Config, printer printer.Printer) error

var queuesMap = map[config.QueueType]queueStorm{
	config.GCPPubSub: runPubSubStorm,
}
