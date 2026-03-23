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

func main() {
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

	if cfg.Queue.Type == config.GCPPubSub && cfg.Connection.PubSub.EmulatorHost != "" {
		err = os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8095")
		if err != nil {
			ptr.Fatal("failed to set PUBSUB_EMULATOR_HOST:", err)
			return
		}
	}
	ptr.Config(cfg)
	pubsubClient, err := gcp.NewClient(context.Background(), "qstorm-project")
	if err != nil {
		ptr.Fatal("pubsub connection failed", err)
		return
	}
	// we check connection status if the emulator host is not set, as emulators doesn't support connection status for pubsub v2.
	if cfg.Connection.PubSub.EmulatorHost == "" {
		err = pubsubClient.Connect(context.Background(), "qstorm-topic")
		if err != nil {
			fmt.Println("failed to connect to pubsub topic: ", err)
			return
		}
	}
	metricCollector := metric.NewCollector()
	start := time.Now()
	res, err := engine.NewEngine(template.NewTemplate(), pubsubClient, metricCollector, ptr).Run(context.Background(), cfg)
	end := time.Since(start)
	if err != nil {
		ptr.Fatal("engine failed with error", err)
	}
	ptr.Summary(res, end)
}
