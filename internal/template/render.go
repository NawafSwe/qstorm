package template

import (
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/nawafswe/qstorm/internal/config"
)

const (
	uuidTemplate      = `\{\{uuid\}\}`
	timestampTemplate = `\{\{timestamp\}\}`
)

type Template struct {
	uuidGen      func() string
	timestampGen func() time.Time
	replacers    map[string]func() string
}

func NewTemplate() Template {
	return Template{
		uuidGen:      func() string { return uuid.NewString() },
		timestampGen: func() time.Time { return time.Now().UTC() },
	}
}

func (t Template) Render(queue config.QueueConfig) (config.QueueConfig, error) {
	// replace all templating values.
	replacers := map[string]func() string{
		uuidTemplate:      t.uuidGen,
		timestampTemplate: func() string { return t.timestampGen().Format(time.RFC3339) },
	}
	for pattern, replacer := range replacers {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return config.QueueConfig{}, err
		}
		queue.Payload = string(re.ReplaceAllFunc([]byte(queue.Payload), func(_ []byte) []byte {
			return []byte(replacer())
		}))
		queue.Attributes = string(re.ReplaceAllFunc([]byte(queue.Attributes), func(_ []byte) []byte {
			return []byte(replacer())
		}))
	}
	return queue, nil
}
