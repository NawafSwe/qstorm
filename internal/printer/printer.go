// Package printer provides k6-style terminal output for load test results.
package printer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/metric"
)

var topicsMapping = map[config.QueueType]func(config.QueueConfig) string{
	config.GCPPubSub: func(c config.QueueConfig) string {
		return fmt.Sprintf("  %stopic%s:     %s", bold, reset, c.PubSub.Topic)
	},
	config.ApacheKafka: func(c config.QueueConfig) string {
		return fmt.Sprintf("  %stopic%s:     %s", bold, reset, c.Kafka.Topic)
	},
	config.ApachePulsar: func(c config.QueueConfig) string {
		return fmt.Sprintf("  %stopic%s:     %s", bold, reset, c.Pulsar.Topic)
	},
	config.RabbitMQ: func(c config.QueueConfig) string {
		s := fmt.Sprintf("  %squeue%s:     %s", bold, reset, c.Rabbitmq.Queue.Name)
		if c.Rabbitmq.Exchange.Name != "" {
			s += fmt.Sprintf("\n  %sexchange%s:  %s", bold, reset, c.Rabbitmq.Exchange.Name)
			s += fmt.Sprintf("\n  %srouting%s:   %s", bold, reset, c.Rabbitmq.Publisher.RoutingKey)
		}
		return s
	},
}

const (
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	reset  = "\033[0m"

	labelWidth = 18
	lineWidth  = 68
)

// Printer handles formatted terminal output styled after k6.
type Printer struct{}

// NewPrinter creates a new Printer.
func NewPrinter() Printer {
	return Printer{}
}

// Banner prints the QStorm ASCII banner.
func (p Printer) Banner() {
	banner := `
      ___  ____  _
     / _ \/ ___|| |_ ___  _ __ _ __ ___
    | | | \___ \| __/ _ \| '__| '_ ` + "`" + ` _ \
    | |_| |___) | || (_) | |  | | | | | |
     \__\_\____/ \__\___/|_|  |_| |_| |_|
`
	fmt.Printf("%s%s%s%s\n", bold, cyan, banner, reset)
}

// Config prints the pre-run configuration summary.
func (p Printer) Config(cfg config.Config) {
	totalDuration := time.Duration(0)
	totalMessages := 0
	for _, s := range cfg.Stages {
		totalDuration += s.Duration
		totalMessages += s.Rate * int(s.Duration.Seconds())
	}

	fmt.Printf("  %sexecution%s: local\n", bold, reset)
	fmt.Printf("  %squeue%s:     %s\n", bold, reset, cfg.Queue.Type)
	printQueueConfig(cfg.Queue)
	fmt.Printf("  %sstages%s:    %d configured, ~%s total\n", bold, reset, len(cfg.Stages), totalDuration)
	fmt.Printf("  %sexpected%s:  ~%d messages\n", bold, reset, totalMessages)
	fmt.Println()

	for i, s := range cfg.Stages {
		fmt.Printf("    %s→%s stage %d: %s @ %d msg/s\n", dim, reset, i+1, s.Duration, s.Rate)
	}

	fmt.Println()
	p.separator()
	fmt.Println()
}

// Progress prints a live progress line that overwrites itself.
func (p Printer) Progress(elapsed time.Duration, currentStage, totalStages int, rate int, published, errors int64) {
	bar := p.progressBar(elapsed)
	fmt.Printf("\r  %s%s%s  stage %d/%d  %d msg/s  %s%d%s pub  %s%d%s err",
		cyan, bar, reset,
		currentStage, totalStages,
		rate,
		green, published, reset,
		red, errors, reset,
	)
}

// Summary prints the post-run metrics summary.
func (p Printer) Summary(s metric.Summary, elapsed time.Duration) {
	// clear the progress line
	fmt.Printf("\r%s\r", strings.Repeat(" ", 80))
	fmt.Println()

	// counts
	p.metricLine("✓", green, "published", fmt.Sprintf("%d", s.SuccessCount))
	p.metricLine("✗", red, "failed", fmt.Sprintf("%d", s.ErrorCount))

	// errors overview
	if len(s.ErrorsOverview) > 0 {
		fmt.Println()
		for msg, count := range s.ErrorsOverview {
			dots := strings.Repeat(".", max(1, labelWidth+10-len(msg)))
			fmt.Printf("         %s%s%s%s: %d\n", red, msg, reset, dots, count)
		}
	}
	fmt.Println()

	// rates
	p.metricLine(" ", "", "success_rate", fmt.Sprintf("%.2f%%", s.SuccessRate))
	p.metricLine(" ", "", "error_rate", fmt.Sprintf("%.2f%%", s.FailureRate))
	fmt.Println()

	// latencies
	p.metricLine(" ", "", "publish_latency", fmt.Sprintf(
		"avg=%s  p50=%s  p75=%s  p90=%s  p99=%s",
		p.fmtMs(s.AverageLatency),
		p.fmtMs(s.P50Latency),
		p.fmtMs(s.P75Latency),
		p.fmtMs(s.P90Latency),
		p.fmtMs(s.P99Latency),
	))
	fmt.Println()

	// total duration
	p.metricLine(" ", "", "duration", elapsed.Truncate(time.Millisecond).String())

	fmt.Println()
	p.separator()
	fmt.Println()
}

// Info prints an informational message.
func (p Printer) Info(msg string) {
	fmt.Printf("  %s●%s %s\n", cyan, reset, msg)
}

// Success prints a success message.
func (p Printer) Success(msg string) {
	fmt.Printf("  %s✓%s %s\n", green, reset, msg)
}

// Warn prints a warning message.
func (p Printer) Warn(msg string) {
	fmt.Printf("  %s⚠%s %s\n", yellow, reset, msg)
}

// Error prints an error message.
func (p Printer) Error(msg string) {
	fmt.Printf("  %s✗%s %s\n", red, reset, msg)
}

// Fatal prints an error message and exits the program.
func (p Printer) Fatal(msg string, err error) {
	fmt.Printf("  %s✗ %s: %v%s\n", red, msg, err, reset)
	os.Exit(1)
}

// PrettyJSON returns an indented JSON string.
func (p Printer) PrettyJSON(v any) string {
	b, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(b)
}

// separator prints a dim horizontal line.
func (p Printer) separator() {
	fmt.Printf("  %s%s%s\n", dim, strings.Repeat("─", lineWidth), reset)
}

// metricLine prints a single metric row: icon + dot-padded label + value.
func (p Printer) metricLine(icon, color, label, value string) {
	dots := strings.Repeat(".", max(1, labelWidth-len(label)))
	if color != "" {
		fmt.Printf("     %s%s%s %s%s%s: %s\n", color, icon, reset, label, dots, dim, value)
	} else {
		fmt.Printf("       %s%s%s: %s\n", label, dots, dim, value)
	}
}

// fmtMs formats a millisecond float as a human-friendly string.
func (p Printer) fmtMs(ms float64) string {
	if ms < 1 {
		return fmt.Sprintf("%.0fµs", ms*1000)
	}
	if ms < 1000 {
		return fmt.Sprintf("%.1fms", ms)
	}
	return fmt.Sprintf("%.2fs", ms/1000)
}

// progressBar builds a visual progress indicator for the current stage.
func (p Printer) progressBar(elapsed time.Duration) string {
	truncated := elapsed.Truncate(time.Second)
	return fmt.Sprintf("running [%s]", truncated)
}

func printQueueConfig(cfg config.QueueConfig) {
	f, found := topicsMapping[cfg.Type]
	if !found {
		fmt.Println()
		return
	}
	fmt.Println(f(cfg))
}
