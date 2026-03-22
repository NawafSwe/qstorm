package template

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nawafswe/qstorm/internal/config"
)

const (
	uuidTemplate      = "{{uuid}}"
	timestampTemplate = "{{timestamp}}"
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
func (t Template) Render(queue config.QueueConfig) (config.QueueConfig, error) {

	// replace all templating values.
	queue.Payload = strings.ReplaceAll(queue.Payload, "{{uuid}}", t.uuidGen())
	queue.Payload = strings.ReplaceAll(queue.Payload, "{{timestamp}}", t.timestampGen().Format(time.RFC3339))

	queue.Attributes = strings.ReplaceAll(queue.Attributes, "{{uuid}}", t.uuidGen())
	queue.Attributes = strings.ReplaceAll(queue.Attributes, "{{timestamp}}", t.timestampGen().Format(time.RFC3339))

	return queue, nil
}
