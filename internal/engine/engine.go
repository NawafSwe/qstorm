package engine

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/messaging"
)

type (
	templateRenderer interface {
		Render(payload string) (string, error)
	}

	messenger interface {
		Publish(ctx context.Context, topic string, message messaging.Message) error
		Close() error
		Connect(ctx context.Context, topic string) error
	}
)

// Result holds the aggregate outcome of a full load test run across all stages.
type Result struct {
	ErrorCounter      int64
	PublishedMessages int64
}

// Engine orchestrates stage execution, publishing messages at a controlled rate.
type Engine struct {
	templateRenderer templateRenderer
	messenger        messenger
	uuidGen          func() string
	timeStampGen     func() time.Time
}

// NewEngine creates and return a new Engine.
func NewEngine(templateRenderer templateRenderer, messenger messenger) Engine {
	return Engine{
		templateRenderer: templateRenderer,
		messenger:        messenger,
		uuidGen:          uuid.NewString,
		timeStampGen: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// Run executes all stages sequentially, accumulating publish results.
func (e Engine) Run(ctx context.Context, request config.Config) (Result, error) {
	var errCounter atomic.Int64
	var publishedCounter atomic.Int64
	var err error
	for _, stage := range request.Stages {
		if err = e.runStage(ctx, request.Queue, stage, &errCounter, &publishedCounter); err != nil {
			break
		}
	}

	return Result{
		ErrorCounter:      errCounter.Load(),
		PublishedMessages: publishedCounter.Load(),
	}, err
}

// runStage publishes messages at the configured rate for the stage's duration.
// Each publish runs in its own goroutine so slow publishes don't block the ticker.
// Returns early with ctx.Err() if the context is cancelled.
func (e Engine) runStage(ctx context.Context, queue config.QueueConfig, stage config.StageConfig, errCounter, publishedMessagesCounter *atomic.Int64) error {
	publishFunc := func(gCtx context.Context) {
		err := e.messenger.Publish(ctx, queue.Topic, messaging.Message{
			ID:          uuid.NewString(),
			Data:        []byte(queue.Payload),
			Attributes:  queue.Attributes,
			OrderingKey: queue.OrderingKey,
		})
		if err != nil {
			errCounter.Add(1)
		} else {
			publishedMessagesCounter.Add(1)
		}
	}
	//  create ticker based on given rate.
	// For example rate=100: 1s / 100 = 10ms per tick — one publish every 10ms = 100 msg/s
	ticker := time.NewTicker(time.Second / time.Duration(stage.Rate))
	defer ticker.Stop()
	deadline := time.After(stage.Duration)
	deadlineCtx, cancel := context.WithDeadline(ctx, time.Now().Add(stage.Duration))
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return nil
		case <-ticker.C:
			go publishFunc(deadlineCtx)
		}
	}
}
