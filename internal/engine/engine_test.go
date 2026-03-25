package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/nawafswe/qstorm/internal/engine/mock"
	"github.com/nawafswe/qstorm/internal/messaging"
	"github.com/nawafswe/qstorm/internal/metric"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestEngine_Run(t *testing.T) {
	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		req        config.Config
		opts       []Option
		ctx        func() (context.Context, context.CancelFunc)
		expectFunc func(
			templateRenderer *mock.MocktemplateRenderer,
			messaging *mock.Mockmessenger,
			metricAggregator *mock.MockmetricAggregator,
			printer *mock.Mockprinter,
		)
		expectedErr error
		want        metric.Summary
	}{
		"single stage publishes messages": {
			req: config.Config{
				Queue: config.QueueConfig{
					PubSub: config.PubSubConfig{
						Topic: "test-topic",
					},
					Type:    config.GCPPubSub,
					Payload: `{"id":"1"}`,
				},
				Stages: []config.StageConfig{
					{Duration: 200 * time.Millisecond, Rate: 10},
				},
			},
			opts: []Option{
				WithUUIDGenerator(func() string { return "fixed-uuid" }),
				WithTimeStampGenerator(func() time.Time { return fixedTime }),
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{
					PubSub: config.PubSubConfig{
						Topic: "test-topic",
					},
					Payload: `{"id":"1"}`,
				}, nil).AnyTimes()

				m.EXPECT().Publish(gomock.Any(), "test-topic", gomock.Any()).Return(nil).AnyTimes()

				ma.EXPECT().Record(gomock.Any(), nil).AnyTimes()
				ma.EXPECT().Snapshot().Return(metric.Snapshot{SuccessCount: 1}).AnyTimes()
				ma.EXPECT().Summary().Return(metric.Summary{SuccessCount: 2, SuccessRate: 100})

				p.EXPECT().Progress(gomock.Any(), 1, 1, 10, gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: metric.Summary{SuccessCount: 2, SuccessRate: 100},
		},

		"multiple stages run sequentially": {
			req: config.Config{
				Queue: config.QueueConfig{
					PubSub: config.PubSubConfig{
						Topic: "test-topic",
					},
					Payload: `{}`,
				},
				Stages: []config.StageConfig{
					{Duration: 150 * time.Millisecond, Rate: 10},
					{Duration: 150 * time.Millisecond, Rate: 20},
				},
			},
			opts: []Option{
				WithUUIDGenerator(func() string { return "fixed-uuid" }),
				WithTimeStampGenerator(func() time.Time { return fixedTime }),
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{
					PubSub: config.PubSubConfig{
						Topic: "test-topic",
					},
					Payload: `{}`,
				}, nil).AnyTimes()

				m.EXPECT().Publish(gomock.Any(), "test-topic", gomock.Any()).Return(nil).AnyTimes()

				ma.EXPECT().Record(gomock.Any(), nil).AnyTimes()
				ma.EXPECT().Snapshot().Return(metric.Snapshot{}).AnyTimes()
				ma.EXPECT().Summary().Return(metric.Summary{SuccessCount: 5, SuccessRate: 100})

				p.EXPECT().Progress(gomock.Any(), gomock.Any(), 2, gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: metric.Summary{SuccessCount: 5, SuccessRate: 100},
		},

		"template render error records metric": {
			req: config.Config{
				Queue: config.QueueConfig{
					PubSub:  config.PubSubConfig{Topic: "t"},
					Payload: `bad`,
				},
				Stages: []config.StageConfig{
					{Duration: 200 * time.Millisecond, Rate: 10},
				},
			},
			opts: []Option{
				WithUUIDGenerator(func() string { return "fixed-uuid" }),
				WithTimeStampGenerator(func() time.Time { return fixedTime }),
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				renderErr := errors.New("invalid template")
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{}, renderErr).AnyTimes()

				ma.EXPECT().Record(time.Duration(0), renderErr).AnyTimes()
				ma.EXPECT().Snapshot().Return(metric.Snapshot{ErrorCount: 1}).AnyTimes()
				ma.EXPECT().Summary().Return(metric.Summary{ErrorCount: 2, FailureRate: 100})

				p.EXPECT().Progress(gomock.Any(), 1, 1, 10, gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: metric.Summary{ErrorCount: 2, FailureRate: 100},
		},

		"publish error records metric": {
			req: config.Config{
				Queue: config.QueueConfig{PubSub: config.PubSubConfig{Topic: "t"}, Payload: `{}`},
				Stages: []config.StageConfig{
					{Duration: 200 * time.Millisecond, Rate: 10},
				},
			},
			opts: []Option{
				WithUUIDGenerator(func() string { return "fixed-uuid" }),
				WithTimeStampGenerator(func() time.Time { return fixedTime }),
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{
					PubSub:  config.PubSubConfig{Topic: "t"},
					Payload: `{}`,
				}, nil).AnyTimes()

				pubErr := errors.New("connection refused")
				m.EXPECT().Publish(gomock.Any(), "t", gomock.Any()).Return(pubErr).AnyTimes()

				ma.EXPECT().Record(gomock.Any(), pubErr).AnyTimes()
				ma.EXPECT().Snapshot().Return(metric.Snapshot{ErrorCount: 1}).AnyTimes()
				ma.EXPECT().Summary().Return(metric.Summary{ErrorCount: 2, FailureRate: 100})

				p.EXPECT().Progress(gomock.Any(), 1, 1, 10, gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: metric.Summary{ErrorCount: 2, FailureRate: 100},
		},

		"context cancellation stops execution": {
			req: config.Config{
				Queue: config.QueueConfig{PubSub: config.PubSubConfig{Topic: "t"}, Payload: `{}`},
				Stages: []config.StageConfig{
					{Duration: 5 * time.Second, Rate: 10},
				},
			},
			opts: []Option{
				WithUUIDGenerator(func() string { return "fixed-uuid" }),
				WithTimeStampGenerator(func() time.Time { return fixedTime }),
			},
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 200*time.Millisecond)
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{
					PubSub:  config.PubSubConfig{Topic: "t"},
					Payload: `{}`,
				}, nil).AnyTimes()

				m.EXPECT().Publish(gomock.Any(), "t", gomock.Any()).Return(nil).AnyTimes()

				ma.EXPECT().Record(gomock.Any(), nil).AnyTimes()
				ma.EXPECT().Snapshot().Return(metric.Snapshot{}).AnyTimes()
				ma.EXPECT().Summary().Return(metric.Summary{})

				p.EXPECT().Progress(gomock.Any(), 1, 1, 10, gomock.Any(), gomock.Any()).AnyTimes()
			},
			expectedErr: context.DeadlineExceeded,
			want:        metric.Summary{},
		},

		"empty stages returns zero summary": {
			req: config.Config{
				Queue:  config.QueueConfig{PubSub: config.PubSubConfig{Topic: "t"}},
				Stages: []config.StageConfig{},
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				ma.EXPECT().Summary().Return(metric.Summary{})
			},
			want: metric.Summary{},
		},

		"default uuid and timestamp generators": {
			req: config.Config{
				Queue: config.QueueConfig{
					PubSub:  config.PubSubConfig{Topic: "t"},
					Payload: `{}`,
				},
				Stages: []config.StageConfig{
					{Duration: 150 * time.Millisecond, Rate: 10},
				},
			},
			// no opts — uses default uuid.NewString and time.Now
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{
					PubSub:  config.PubSubConfig{Topic: "t"},
					Payload: `{}`,
				}, nil).AnyTimes()

				m.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				ma.EXPECT().Record(gomock.Any(), gomock.Any()).AnyTimes()
				ma.EXPECT().Snapshot().Return(metric.Snapshot{}).AnyTimes()
				ma.EXPECT().Summary().Return(metric.Summary{SuccessCount: 1, SuccessRate: 100})

				p.EXPECT().Progress(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: metric.Summary{SuccessCount: 1, SuccessRate: 100},
		},

		"nil option generators keep defaults": {
			req: config.Config{
				Queue: config.QueueConfig{
					PubSub:  config.PubSubConfig{Topic: "t"},
					Payload: `{}`,
				},
				Stages: []config.StageConfig{
					{Duration: 150 * time.Millisecond, Rate: 10},
				},
			},
			opts: []Option{
				WithUUIDGenerator(nil),
				WithTimeStampGenerator(nil),
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{
					PubSub:  config.PubSubConfig{Topic: "t"},
					Payload: `{}`,
				}, nil).AnyTimes()

				m.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				ma.EXPECT().Record(gomock.Any(), gomock.Any()).AnyTimes()
				ma.EXPECT().Snapshot().Return(metric.Snapshot{}).AnyTimes()
				ma.EXPECT().Summary().Return(metric.Summary{SuccessCount: 1, SuccessRate: 100})

				p.EXPECT().Progress(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: metric.Summary{SuccessCount: 1, SuccessRate: 100},
		},

		"progress prints snapshot during stage": {
			req: config.Config{
				Queue: config.QueueConfig{PubSub: config.PubSubConfig{Topic: "t"}, Payload: `{}`},
				Stages: []config.StageConfig{
					{Duration: 150 * time.Millisecond, Rate: 100},
				},
			},
			opts: []Option{
				WithUUIDGenerator(func() string { return "id" }),
				WithTimeStampGenerator(func() time.Time { return fixedTime }),
				WithProgressTicker(30 * time.Millisecond),
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{
					PubSub: config.PubSubConfig{Topic: "t"}, Payload: `{}`,
				}, nil).AnyTimes()

				m.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				ma.EXPECT().Record(gomock.Any(), gomock.Any()).AnyTimes()

				ma.EXPECT().Snapshot().Return(metric.Snapshot{SuccessCount: 5, ErrorCount: 0}).MinTimes(1)
				p.EXPECT().Progress(gomock.Any(), 1, 1, 100, int64(5), int64(0)).MinTimes(1)

				ma.EXPECT().Summary().Return(metric.Summary{SuccessCount: 10, SuccessRate: 100})
			},
			want: metric.Summary{SuccessCount: 10, SuccessRate: 100},
		},

		"publishes correct message fields": {
			req: config.Config{
				Queue: config.QueueConfig{
					PubSub:     config.PubSubConfig{Topic: "my-topic", OrderingKey: "order-123"},
					Payload:    `{"key":"val"}`,
					Attributes: `{"attr":"1"}`,
				},
				Stages: []config.StageConfig{
					{Duration: 150 * time.Millisecond, Rate: 10},
				},
			},
			opts: []Option{
				WithUUIDGenerator(func() string { return "test-uuid" }),
				WithTimeStampGenerator(func() time.Time { return fixedTime }),
			},
			expectFunc: func(tr *mock.MocktemplateRenderer, m *mock.Mockmessenger, ma *mock.MockmetricAggregator, p *mock.Mockprinter) {
				tr.EXPECT().Render(gomock.Any()).Return(config.QueueConfig{
					PubSub:     config.PubSubConfig{Topic: "my-topic", OrderingKey: "order-123"},
					Payload:    `{"key":"val"}`,
					Attributes: `{"attr":"1"}`,
				}, nil).AnyTimes()

				m.EXPECT().Publish(gomock.Any(), "my-topic", messaging.Message{
					ID:          "test-uuid",
					Data:        []byte(`{"key":"val"}`),
					Attributes:  `{"attr":"1"}`,
					OrderingKey: "order-123",
				}).Return(nil).AnyTimes()

				ma.EXPECT().Record(gomock.Any(), nil).AnyTimes()
				ma.EXPECT().Snapshot().Return(metric.Snapshot{}).AnyTimes()
				ma.EXPECT().Summary().Return(metric.Summary{SuccessCount: 1, SuccessRate: 100})

				p.EXPECT().Progress(gomock.Any(), 1, 1, 10, gomock.Any(), gomock.Any()).AnyTimes()
			},
			want: metric.Summary{SuccessCount: 1, SuccessRate: 100},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			templateRendererMock := mock.NewMocktemplateRenderer(ctrl)
			messagingMock := mock.NewMockmessenger(ctrl)
			metricAggregatorMock := mock.NewMockmetricAggregator(ctrl)
			printerMock := mock.NewMockprinter(ctrl)
			if tc.expectFunc != nil {
				tc.expectFunc(templateRendererMock, messagingMock, metricAggregatorMock, printerMock)
			}

			e := NewEngine(templateRendererMock, messagingMock, metricAggregatorMock, printerMock, tc.opts...)

			ctx := context.Background()
			var cancel context.CancelFunc
			if tc.ctx != nil {
				ctx, cancel = tc.ctx()
				defer cancel()
			}

			res, err := e.Run(ctx, tc.req)

			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, res)
		})
	}
}
