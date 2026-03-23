// Package template handles payload and attribute variable substitution.
package template

import (
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/nawafswe/qstorm/internal/config"
)

// Pre-compiled patterns to avoid re-compiling on every Render call.
var (
	uuidPattern      = regexp.MustCompile(`\{\{uuid\}\}`)
	timestampPattern = regexp.MustCompile(`\{\{timestamp\}\}`)
)

// Option configures optional Template behavior.
type Option func(*Template)

// WithUUIDGenerator overrides the default UUID generator.
func WithUUIDGenerator(gen func() string) Option {
	return func(t *Template) {
		if gen != nil {
			t.uuidGen = gen
		}
	}
}

// WithTimestampGenerator overrides the default timestamp generator.
func WithTimestampGenerator(gen func() time.Time) Option {
	return func(t *Template) {
		if gen != nil {
			t.timestampGen = gen
		}
	}
}

// Template holds generators for template variable substitution.
type Template struct {
	uuidGen      func() string
	timestampGen func() time.Time
}

// NewTemplate creates a Template with default UUID and timestamp generators.
func NewTemplate(opts ...Option) Template {
	t := Template{
		uuidGen:      func() string { return uuid.NewString() },
		timestampGen: func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(&t)
	}
	return t
}

// Render replaces template variables in Payload and Attributes.
// Each {{uuid}} gets a unique value; each {{timestamp}} gets the current time.
func (t Template) Render(queue config.QueueConfig) (config.QueueConfig, error) {
	replacers := map[*regexp.Regexp]func() string{
		uuidPattern:      t.uuidGen,
		timestampPattern: func() string { return t.timestampGen().Format(time.RFC3339) },
	}
	for re, gen := range replacers {
		queue.Payload = string(re.ReplaceAllFunc([]byte(queue.Payload), func(_ []byte) []byte {
			return []byte(gen())
		}))
		queue.Attributes = string(re.ReplaceAllFunc([]byte(queue.Attributes), func(_ []byte) []byte {
			return []byte(gen())
		}))
	}
	return queue, nil
}
