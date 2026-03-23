package template

import (
	"testing"
	"time"

	"github.com/nawafswe/qstorm/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestTemplate_Render(t *testing.T) {
	fixedTime := time.Date(2026, 3, 23, 14, 30, 0, 0, time.UTC)

	tests := map[string]struct {
		opts       []Option
		queue      config.QueueConfig
		want       config.QueueConfig
		assertFunc func(t *testing.T, result config.QueueConfig)
	}{
		"replaces uuid in payload": {
			opts: []Option{
				WithUUIDGenerator(func() string { return "test-uuid" }),
				WithTimestampGenerator(func() time.Time { return fixedTime }),
			},
			queue: config.QueueConfig{
				Payload: `{"id":"{{uuid}}"}`,
			},
			want: config.QueueConfig{
				Payload: `{"id":"test-uuid"}`,
			},
		},

		"replaces timestamp in payload": {
			opts: []Option{
				WithUUIDGenerator(func() string { return "u" }),
				WithTimestampGenerator(func() time.Time { return fixedTime }),
			},
			queue: config.QueueConfig{
				Payload: `{"ts":"{{timestamp}}"}`,
			},
			want: config.QueueConfig{
				Payload: `{"ts":"2026-03-23T14:30:00Z"}`,
			},
		},

		"replaces uuid in attributes": {
			opts: []Option{
				WithUUIDGenerator(func() string { return "attr-uuid" }),
				WithTimestampGenerator(func() time.Time { return fixedTime }),
			},
			queue: config.QueueConfig{
				Attributes: `{"id":"{{uuid}}"}`,
			},
			want: config.QueueConfig{
				Attributes: `{"id":"attr-uuid"}`,
			},
		},

		"replaces timestamp in attributes": {
			opts: []Option{
				WithUUIDGenerator(func() string { return "u" }),
				WithTimestampGenerator(func() time.Time { return fixedTime }),
			},
			queue: config.QueueConfig{
				Attributes: `{"ts":"{{timestamp}}"}`,
			},
			want: config.QueueConfig{
				Attributes: `{"ts":"2026-03-23T14:30:00Z"}`,
			},
		},

		"multiple uuids get unique values": {
			opts: []Option{
				WithUUIDGenerator(func() func() string {
					count := 0
					return func() string {
						count++
						return "uuid-" + string(rune('0'+count))
					}
				}()),
				WithTimestampGenerator(func() time.Time { return fixedTime }),
			},
			queue: config.QueueConfig{
				Payload: `{"a":"{{uuid}}","b":"{{uuid}}"}`,
			},
			want: config.QueueConfig{
				Payload: `{"a":"uuid-1","b":"uuid-2"}`,
			},
		},

		"no template variables unchanged": {
			opts: []Option{
				WithUUIDGenerator(func() string { return "should-not-appear" }),
				WithTimestampGenerator(func() time.Time { return fixedTime }),
			},
			queue: config.QueueConfig{
				Payload:    `{"static":"value"}`,
				Attributes: `{"key":"val"}`,
			},
			want: config.QueueConfig{
				Payload:    `{"static":"value"}`,
				Attributes: `{"key":"val"}`,
			},
		},

		"empty payload and attributes": {
			opts: []Option{
				WithUUIDGenerator(func() string { return "u" }),
				WithTimestampGenerator(func() time.Time { return fixedTime }),
			},
			queue: config.QueueConfig{},
			want:  config.QueueConfig{},
		},

		"preserves non-template fields": {
			opts: []Option{
				WithUUIDGenerator(func() string { return "u" }),
				WithTimestampGenerator(func() time.Time { return fixedTime }),
			},
			queue: config.QueueConfig{
				Topic:       "my-topic",
				OrderingKey: "key-1",
				Type:        config.GCPPubSub,
				Payload:     `{{uuid}}`,
			},
			want: config.QueueConfig{
				Topic:       "my-topic",
				OrderingKey: "key-1",
				Type:        config.GCPPubSub,
				Payload:     "u",
			},
		},
		"default generators produce valid output": {
			queue: config.QueueConfig{
				Payload:    `{"id":"{{uuid}}","ts":"{{timestamp}}"}`,
				Attributes: `{"aid":"{{uuid}}"}`,
			},
			assertFunc: func(t *testing.T, result config.QueueConfig) {
				assert.NotContains(t, result.Payload, "{{uuid}}")
				assert.NotContains(t, result.Payload, "{{timestamp}}")
				assert.NotContains(t, result.Attributes, "{{uuid}}")
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tmpl := NewTemplate(tc.opts...)
			result, err := tmpl.Render(tc.queue)

			assert.NoError(t, err)
			if tc.assertFunc != nil {
				tc.assertFunc(t, result)
			} else {
				assert.Equal(t, tc.want, result)
			}
		})
	}
}
