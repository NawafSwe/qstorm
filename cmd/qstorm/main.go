package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/printer"
)

func main() {
	configPath := flag.String("config", "", "path to the JSON test config file")
	envPath := flag.String("env", ".env", "path to the .env connection file")
	flag.Parse()

	if *configPath == "" && flag.NArg() > 0 {
		*configPath = flag.Arg(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ptr := printer.NewPrinter()
	ptr.Banner()

	if *configPath == "" {
		ptr.Fatal("usage", fmt.Errorf("qstorm --config <config.json> [--env <.env>]"))
	}

	cfg, err := config.LoadJSONConfig(*configPath)
	if err != nil {
		ptr.Fatal("failed to load config", err)
	}
	connEnv, err := config.LoadConnConfig(*envPath)
	if err != nil {
		ptr.Fatal("failed to load connection config", err)
	}
	cfg.Connection = connEnv
	ptr.Config(cfg)

	storm, ok := queuesMap[cfg.Queue.Type]
	if !ok {
		ptr.Fatal("unsupported queue type", fmt.Errorf("%q — available: %s", cfg.Queue.Type, availableQueues()))
	}
	err = storm(queueArgs{
		ctx:     ctx,
		cfg:     cfg,
		printer: ptr,
	}).Run(ctx)
	if err != nil && ctx.Err() == nil {
		ptr.Fatal("failed to run queue", err)
	}
}

func availableQueues() string {
	names := make([]string, 0, len(queuesMap))
	for k := range queuesMap {
		names = append(names, string(k))
	}
	return strings.Join(names, ", ")
}
