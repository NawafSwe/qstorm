package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/engine"
	"github.com/nawafswe/qstorm/internal/messaging/pulsar"
	"github.com/nawafswe/qstorm/internal/metric"
	"github.com/nawafswe/qstorm/internal/printer"
	"github.com/nawafswe/qstorm/internal/template"
)

type pulsarStorm struct {
	cfg     config.Config
	printer printer.Printer
}

func (p pulsarStorm) Run(ctx context.Context) error {
	if err := p.Validate(); err != nil {
		return err
	}
	return p.storm(ctx)
}

func (p pulsarStorm) Validate() error {
	if p.cfg.Connection.Pulsar.URL == "" {
		return fmt.Errorf("pulsar URL is required")
	}
	return nil
}

func (p pulsarStorm) storm(ctx context.Context) error {
	pulsarClient, err := pulsar.NewClient(ctx, p.cfg.Queue.Pulsar, p.cfg.Connection.Pulsar)
	if err != nil {
		return fmt.Errorf("pulsar client connection failed: %w", err)
	}
	metricCollector := metric.NewCollector()
	start := time.Now()
	rabbitmqStormEngine := engine.NewEngine(template.NewTemplate(), pulsarClient, metricCollector, p.printer)
	res, err := rabbitmqStormEngine.Run(ctx, p.cfg)
	end := time.Since(start)
	p.printer.Summary(res, end)
	if err != nil {
		return fmt.Errorf("pulsar engine failed with error: %w", err)
	}

	return nil
}
