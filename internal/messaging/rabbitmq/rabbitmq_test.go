package rabbitmq

import (
	"fmt"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func Test_headers(t *testing.T) {
	tests := map[string]struct {
		attrs       string
		want        amqp.Table
		expectedErr error
	}{
		"should successfully parse message attributes": {
			attrs: `{"SOURCE":"qstorm-test"}`,
			want:  amqp.Table{"SOURCE": "qstorm-test"},
		},
		"should return empty headers when no attributes passed": {
			attrs: ``,
		},
		"should fail parsing message attributes when invalid json string passed": {
			attrs:       `{"SOURCE":"qstorm-test#`,
			expectedErr: fmt.Errorf("failed to unmarshal message attributes: unexpected end of JSON input"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := headers(tc.attrs)
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
