package kafka

import (
	"fmt"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/assert"
)

func Test_messageHeaders(t *testing.T) {
	tests := map[string]struct {
		attrs       string
		want        []kafka.Header
		expectedErr error
	}{
		"should successfully parse message attributes": {
			attrs: `{"SOURCE":"qstorm-test"}`,
			want: []kafka.Header{
				{
					Key:   "SOURCE",
					Value: []byte("qstorm-test"),
				},
			},
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

			got, err := messageHeaders(tc.attrs)
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
