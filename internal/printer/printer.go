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

const (
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	reset  = "\033[0m"

	metricWidth = 22
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
	fmt.Printf("  %sexecution%s: local\n", bold, reset)
	fmt.Printf("  %squeue%s:     %s\n", bold, reset, cfg.Queue.Type)
	fmt.Printf("  %stopic%s:     %s\n", bold, reset, cfg.Queue.Topic)
	fmt.Println()

	totalDuration := time.Duration(0)
	totalMessages := 0
	for _, s := range cfg.Stages {
		totalDuration += s.Duration
		totalMessages += s.Rate * int(s.Duration.Seconds())
	}

	fmt.Printf("  %sstages%s:    %d configured, ~%s total duration\n", bold, reset, len(cfg.Stages), totalDuration)
	fmt.Printf("  %sexpected%s:  ~%d messages\n", bold, reset, totalMessages)
	fmt.Println()

	for i, s := range cfg.Stages {
		fmt.Printf("    %s→%s stage %d: %s @ %d msg/s\n", dim, reset, i+1, s.Duration, s.Rate)
	}
	fmt.Println()
}

// Progress prints a live progress line that overwrites itself.
func (p Printer) Progress(elapsed time.Duration, currentStage, totalStages int, rate int, published, errors int64) {
	fmt.Printf("\r  %srunning%s (%s), stage %d/%d, %d msg/s, %s%d%s published, %s%d%s failed",
		cyan, reset,
		elapsed.Truncate(time.Millisecond),
		currentStage, totalStages,
		rate,
		green, published, reset,
		red, errors, reset,
	)
}

// Summary prints the post-run metrics summary.
func (p Printer) Summary(s metric.Summary, elapsed time.Duration) {
	fmt.Println()
	fmt.Println()

	// published / failed
	if s.SuccessCount > 0 {
		fmt.Printf("     %s✓%s %s\n", green, reset, p.metric("published", fmt.Sprintf("%d", s.SuccessCount)))
	}
	if s.ErrorCount > 0 {
		fmt.Printf("     %s✗%s %s\n", red, reset, p.metric("failed", fmt.Sprintf("%d", s.ErrorCount)))
	}
	fmt.Println()

	// rates
	fmt.Printf("     %s\n", p.metric("success_rate", fmt.Sprintf("%.2f%%", s.SuccessRate)))
	fmt.Printf("     %s\n", p.metric("error_rate", fmt.Sprintf("%.2f%%", s.FailureRate)))
	fmt.Println()

	// latencies
	fmt.Printf("     %s\n", p.metric("publish_lat",
		fmt.Sprintf("avg=%.2fms  p50=%.2fms  p75=%.2fms  p90=%.2fms  p99=%.2fms",
			s.AverageLatency, s.P50Latency, s.P75Latency, s.P90Latency, s.P99Latency)))
	fmt.Println()

	// duration
	fmt.Printf("     %s\n", p.metric("duration", elapsed.Truncate(time.Millisecond).String()))
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

// metric formats a label with dot-padding and a value, k6 style.
func (p Printer) metric(label, value string) string {
	dots := strings.Repeat(".", max(1, metricWidth-len(label)))
	return fmt.Sprintf("%s%s: %s", label, dots, value)
}
