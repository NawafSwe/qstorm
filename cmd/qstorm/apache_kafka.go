package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/engine"
	"github.com/nawafswe/qstorm/internal/messaging/kafka"
	"github.com/nawafswe/qstorm/internal/metric"
	"github.com/nawafswe/qstorm/internal/printer"
	"github.com/nawafswe/qstorm/internal/template"
)

type kafkaStorm struct {
	cfg     config.Config
	printer printer.Printer
}

func (p kafkaStorm) Run(ctx context.Context) error {
	if err := p.Validate(); err != nil {
		return err
	}
	return p.storm(ctx)
}

func (p kafkaStorm) Validate() error {
	if p.cfg.Connection.Kafka.BootstrapServers == "" {
		return fmt.Errorf("kafka bootstrap servers is required")
	}
	return nil
}

func (p kafkaStorm) storm(ctx context.Context) error {
	kafkaClient, err := kafka.NewClient(ctx, p.cfg.Connection.Kafka, p.cfg.Queue.Kafka)
	if err != nil {
		return fmt.Errorf("kafka client connection failed: %w", err)
	}
	metricCollector := metric.NewCollector()
	start := time.Now()
	kafkaStormEngine := engine.NewEngine(template.NewTemplate(), kafkaClient, metricCollector, p.printer)
	res, err := kafkaStormEngine.Run(ctx, p.cfg)
	end := time.Since(start)
	p.printer.Summary(res, end)
	if err != nil {
		return fmt.Errorf("kafka engine failed with error: %w", err)
	}

	return nil
}
