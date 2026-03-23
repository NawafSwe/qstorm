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

type pubSubStorm struct {
	cfg     config.Config
	printer printer.Printer
}

func (p pubSubStorm) Run(ctx context.Context) error {
	if err := p.Validate(); err != nil {
		return err
	}
	return p.storm(ctx)
}

func (p pubSubStorm) Validate() error {
	if p.cfg.Connection.PubSub.ProjectID == "" {
		return fmt.Errorf("pubsub project id is required")
	}
	return nil
}

func (p pubSubStorm) storm(ctx context.Context) error {
	if p.cfg.Connection.PubSub.EmulatorHost != "" {
		err := os.Setenv("PUBSUB_EMULATOR_HOST", p.cfg.Connection.PubSub.EmulatorHost)
		if err != nil {
			return fmt.Errorf("failed to set PUBSUB_EMULATOR_HOST: %w", err)
		}
	}
	var opts []gcp.Option
	if pubsubCreds := p.cfg.Connection.PubSub.CredentialsFile; pubsubCreds != "" {
		creds := pubsubCreds.String()
		opts = append(opts, gcp.WithServiceAccountCredentials(&creds))
	}
	pubsubClient, err := gcp.NewClient(context.Background(), p.cfg.Connection.PubSub.ProjectID, opts...)
	if err != nil {
		return fmt.Errorf("pubsub connection failed: %w", err)
	}
	// we check connection status if the emulator host is not set, as emulators doesn't support connection status for pubsub v2.
	if p.cfg.Connection.PubSub.EmulatorHost == "" {
		err = pubsubClient.Connect(context.Background(), "qstorm-topic")
		if err != nil {
			return fmt.Errorf("failed to connect to pubsub topic: %w", err)
		}
	}
	metricCollector := metric.NewCollector()
	start := time.Now()
	pubsubStormEngine := engine.NewEngine(template.NewTemplate(), pubsubClient, metricCollector, p.printer)
	res, err := pubsubStormEngine.Run(ctx, p.cfg)
	end := time.Since(start)
	p.printer.Summary(res, end)
	if err != nil {
		return fmt.Errorf("pubsub engine failed with error: %w", err)
	}

	return nil
}
