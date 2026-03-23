package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/printer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ptr := printer.NewPrinter()

	ptr.Banner()

	cfg, err := config.LoadJSONConfig("example/gcp_pubsub_test_config.json")
	if err != nil {
		ptr.Fatal("failed to load config", err)
	}
	connEnv, err := config.LoadConnConfigFromEnv("./.env")
	if err != nil {
		ptr.Fatal("failed to load connection config", err)
	}
	cfg.Connection = connEnv
	ptr.Config(cfg)

	queueFunc, ok := queuesMap[cfg.Queue.Type]
	if !ok {
		ptr.Fatal("queue type not supported", nil)
	}
	err = queueFunc(ctx, cfg, ptr)
	// if not ctx err.
	if err != nil && ctx.Err() == nil {
		ptr.Fatal("failed to run queue", err)
	}
}
