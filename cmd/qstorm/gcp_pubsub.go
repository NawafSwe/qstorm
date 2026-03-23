package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/engine"
	"github.com/nawafswe/qstorm/internal/messaging/gcp"
	"github.com/nawafswe/qstorm/internal/metric"
	"github.com/nawafswe/qstorm/internal/printer"
	"github.com/nawafswe/qstorm/internal/template"
)

func runPubSubStorm(ctx context.Context, cfg config.Config, ptr printer.Printer) error {
	if cfg.Queue.Type == config.GCPPubSub && cfg.Connection.PubSub.EmulatorHost != "" {
		err := os.Setenv("PUBSUB_EMULATOR_HOST", cfg.Connection.PubSub.EmulatorHost)
		if err != nil {
			return fmt.Errorf("failed to set PUBSUB_EMULATOR_HOST: %w", err)
		}
	}
	pubsubClient, err := gcp.NewClient(context.Background(), cfg.Connection.PubSub.ProjectID)
	if err != nil {
		return fmt.Errorf("pubsub connection failed: %w", err)
	}
	// we check connection status if the emulator host is not set, as emulators doesn't support connection status for pubsub v2.
	if cfg.Connection.PubSub.EmulatorHost == "" {
		err = pubsubClient.Connect(context.Background(), "qstorm-topic")
		if err != nil {
			return fmt.Errorf("failed to connect to pubsub topic: %w", err)
		}
	}
	metricCollector := metric.NewCollector()
	start := time.Now()
	pubsubStormEngine := engine.NewEngine(template.NewTemplate(), pubsubClient, metricCollector, ptr)
	res, err := pubsubStormEngine.Run(ctx, cfg)
	end := time.Since(start)
	if err != nil {
		return fmt.Errorf("pubsub engine failed with error: %w", err)
	}
	ptr.Summary(res, end)
	return nil
}
