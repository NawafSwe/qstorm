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

type Template struct {
	uuidGen      func() string
	timestampGen func() time.Time
}

func NewTemplate() Template {
	return Template{
		uuidGen:      func() string { return uuid.NewString() },
		timestampGen: func() time.Time { return time.Now().UTC() },
	}
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
