package metric

import (
	"sync"
	"sync/atomic"
	"time"

	hd "github.com/HdrHistogram/hdrhistogram-go"
)

const (
	minLatencyMicros = 1
	maxLatencyMicros = 60_000_000
	histogramSigFigs = 3

	p99PercentileValue = 99
	p90PercentileValue = 90
	p75PercentileValue = 75
	p50PercentileValue = 50
	microsPerMS        = 1000.0
)

type (
	// Summary encapsulates the summary of metric collection.
	Summary struct {
		ErrorCount   int64
		SuccessCount int64

		SuccessRate float64
		FailureRate float64

		// AverageLatency in milliseconds.
		AverageLatency float64

		// P99Latency in milliseconds.
		P99Latency float64

		// P90Latency in milliseconds.
		P90Latency float64

		// P75Latency in milliseconds.
		P75Latency float64

		// P50Latency in milliseconds.
		P50Latency float64
	}
	// Snapshot is a snapshot of the metric collection.
	Snapshot struct {
		ErrorCount   int64
		SuccessCount int64
	}
)
type Collector struct {
	mu           sync.Mutex
	errorCount   atomic.Int64
	successCount atomic.Int64
	histogram    *hd.Histogram
}

func NewCollector() *Collector {
	return &Collector{
		mu:        sync.Mutex{},
		histogram: hd.New(minLatencyMicros, maxLatencyMicros, histogramSigFigs),
	}
}

func (c *Collector) Summary() Summary {
	c.mu.Lock()
	defer c.mu.Unlock()

	errors := c.errorCount.Load()
	successes := c.successCount.Load()
	total := errors + successes

	var successRate, failureRate float64
	if total > 0 {
		successRate = float64(successes) / float64(total) * 100
		failureRate = float64(errors) / float64(total) * 100
	}

	return Summary{
		ErrorCount:     errors,
		SuccessCount:   successes,
		SuccessRate:    successRate,
		FailureRate:    failureRate,
		AverageLatency: c.histogram.Mean() / microsPerMS,
		P99Latency:     float64(c.histogram.ValueAtPercentile(p99PercentileValue)) / microsPerMS,
		P90Latency:     float64(c.histogram.ValueAtPercentile(p90PercentileValue)) / microsPerMS,
		P75Latency:     float64(c.histogram.ValueAtPercentile(p75PercentileValue)) / microsPerMS,
		P50Latency:     float64(c.histogram.ValueAtPercentile(p50PercentileValue)) / microsPerMS,
	}
}

func (c *Collector) Record(executionTime time.Duration, encounteredErr error) {
	if encounteredErr != nil {
		c.errorCount.Add(1)
		return
	}

	c.successCount.Add(1)
	c.mu.Lock()

	// ignoring the error as there is no need to block the execution process.
	_ = c.histogram.RecordValue(executionTime.Microseconds())

	c.mu.Unlock()
}

// Snapshot returns a snapshot of the metric collection.
func (c *Collector) Snapshot() Snapshot {
	return Snapshot{
		ErrorCount:   c.errorCount.Load(),
		SuccessCount: c.successCount.Load(),
	}
}
