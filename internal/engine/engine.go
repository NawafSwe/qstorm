package engine

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/messaging"
	"github.com/nawafswe/qstorm/internal/metric"
)

//go:generate go tool mockgen -source=${GOFILE} -destination=mock/${GOFILE} -package=mock
type (
	templateRenderer interface {
		Render(queue config.QueueConfig) (config.QueueConfig, error)
	}

	messenger interface {
		Publish(ctx context.Context, topic string, message messaging.Message) error
		Close() error
		Connect(ctx context.Context, topic string) error
	}
	metricAggregator interface {
		Record(executionTime time.Duration, encounteredErr error)
		Summary() metric.Summary
		Snapshot() metric.Snapshot
	}
	printer interface {
		Progress(elapsed time.Duration, currentStage, totalStages int, rate int, published, errors int64)
	}
)

// Engine orchestrates stage execution, publishing messages at a controlled rate.
type Engine struct {
	templateRenderer templateRenderer
	messenger        messenger
	printer          printer
	metricAggregator metricAggregator
	uuidGen          func() string
	timeStampGen     func() time.Time
}

// NewEngine creates and return a new Engine.
func NewEngine(templateRenderer templateRenderer, messenger messenger, metricAggregator metricAggregator, printer printer) Engine {
	return Engine{
		templateRenderer: templateRenderer,
		messenger:        messenger,
		uuidGen:          uuid.NewString,
		timeStampGen: func() time.Time {
			return time.Now().UTC()
		},
		metricAggregator: metricAggregator,
		printer:          printer,
	}
}

// Run executes all stages sequentially, accumulating publish results.
func (e Engine) Run(ctx context.Context, request config.Config) (metric.Summary, error) {
	var err error
	totalStages := len(request.Stages)
	for i, stage := range request.Stages {
		currentStage := i + 1
		if err = e.runStage(ctx, request.Queue, stage, currentStage, totalStages); err != nil {
			break
		}
	}

	return e.metricAggregator.Summary(), err
}

// runStage publishes messages at the configured rate for the stage's duration.
// Each publish runs in its own goroutine so slow publishes don't block the ticker.
// Returns early with ctx.Err() if the context is cancelled.
func (e Engine) runStage(ctx context.Context, queue config.QueueConfig, stage config.StageConfig, currentStage, totalStages int) error {
	publishFunc := func(gCtx context.Context) {
		templatedQueue, err := e.templateRenderer.Render(queue)
		if err != nil {
			e.metricAggregator.Record(0, err)
			return
		}

		start := e.timeStampGen()
		err = e.messenger.Publish(gCtx, templatedQueue.Topic, messaging.Message{
			ID:          e.uuidGen(),
			Data:        []byte(templatedQueue.Payload),
			Attributes:  templatedQueue.Attributes,
			OrderingKey: templatedQueue.OrderingKey,
		})
		end := time.Since(start)
		e.metricAggregator.Record(end, err)
	}
	//  create ticker based on given rate.
	// For example rate=100: 1s / 100 = 10ms per tick — one publish every 10ms = 100 msg/s
	ticker := time.NewTicker(time.Second / time.Duration(stage.Rate))
	defer ticker.Stop()

	progressTicker := time.NewTicker(600 * time.Millisecond)
	defer progressTicker.Stop()

	deadline := time.After(stage.Duration)
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(stage.Duration))
	defer cancel()

	stageStart := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return nil
		case <-ticker.C:
			go publishFunc(deadlineCtx)
		case <-progressTicker.C:
			snapshot := e.metricAggregator.Snapshot()
			e.printer.Progress(time.Since(stageStart), currentStage, totalStages, stage.Rate, snapshot.SuccessCount, snapshot.ErrorCount)
		}
	}
}
