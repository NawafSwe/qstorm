package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/engine"
	"github.com/nawafswe/qstorm/internal/messaging/rabbitmq"
	"github.com/nawafswe/qstorm/internal/metric"
	"github.com/nawafswe/qstorm/internal/printer"
	"github.com/nawafswe/qstorm/internal/template"
)

type rabbitmqStorm struct {
	cfg     config.Config
	printer printer.Printer
}

func (p rabbitmqStorm) Run(ctx context.Context) error {
	if err := p.Validate(); err != nil {
		return err
	}
	return p.storm(ctx)
}

func (p rabbitmqStorm) Validate() error {
	if p.cfg.Connection.Rabbitmq.URL == "" {
		return fmt.Errorf("rabbitmq URL is required")
	}
	return nil
}

func (p rabbitmqStorm) storm(ctx context.Context) error {
	rabbitmqClient, err := rabbitmq.NewClient(ctx, p.cfg.Connection.Rabbitmq.URL, p.cfg.Queue.Rabbitmq)
	if err != nil {
		return fmt.Errorf("rabbitmq client connection failed: %w", err)
	}
	metricCollector := metric.NewCollector()
	start := time.Now()
	rabbitmqStormEngine := engine.NewEngine(template.NewTemplate(), rabbitmqClient, metricCollector, p.printer)
	res, err := rabbitmqStormEngine.Run(ctx, p.cfg)
	end := time.Since(start)
	p.printer.Summary(res, end)
	if err != nil {
		return fmt.Errorf("rabbitmq engine failed with error: %w", err)
	}

	return nil
}
