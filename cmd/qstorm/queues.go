package main

import (
	"context"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/printer"
)

type queueArgs struct {
	ctx     context.Context
	cfg     config.Config
	printer printer.Printer
}
type stormer interface {
	Run(ctx context.Context) error
	Validate() error
}

var queuesMap = map[config.QueueType]func(args queueArgs) stormer{
	config.GCPPubSub: func(args queueArgs) stormer {
		return pubSubStorm{
			cfg:     args.cfg,
			printer: args.printer,
		}
	},
	config.ApacheKafka: func(args queueArgs) stormer {
		return kafkaStorm{
			cfg:     args.cfg,
			printer: args.printer,
		}
	},
}
