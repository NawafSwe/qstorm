package pulsar

import (
	"fmt"
	"testing"

	apacheplsr "github.com/apache/pulsar-client-go/pulsar"
	"github.com/nawafswe/qstorm/internal/config"
	"github.com/stretchr/testify/assert"
)

func Test_messageProperties(t *testing.T) {
	tests := map[string]struct {
		attrs       string
		want        map[string]string
		expectedErr error
	}{
		"should successfully parse message properties": {
			attrs: `{"SOURCE":"qstorm-test","ENV":"integration"}`,
			want: map[string]string{
				"SOURCE": "qstorm-test",
				"ENV":    "integration",
			},
		},
		"should return nil when no attributes passed": {
			attrs: ``,
		},
		"should fail parsing when invalid json string passed": {
			attrs:       `{"SOURCE":"qstorm-test#`,
			expectedErr: fmt.Errorf("failed to unmarshal message properties: unexpected end of JSON input"),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := messageProperties(tc.attrs)
			if tc.expectedErr != nil {
				assert.EqualError(t, err, tc.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func Test_auth(t *testing.T) {
	tests := map[string]struct {
		cfg      config.PulsarConnectionConfig
		wantNil  bool
		wantType string
	}{
		"returns nil when no auth configured": {
			cfg:     config.PulsarConnectionConfig{URL: "pulsar://localhost:6650"},
			wantNil: true,
		},
		"returns token auth when token is set": {
			cfg: config.PulsarConnectionConfig{
				URL:       "pulsar://localhost:6650",
				AuthToken: strPtr("my-jwt-token"),
			},
		},
		"returns basic auth when basic auth is set": {
			cfg: config.PulsarConnectionConfig{
				URL: "pulsar://localhost:6650",
				BasicAuth: &config.BasicAuth{
					Username: "admin",
					Password: config.NonLoggable("admin-pass"),
				},
			},
		},
		"token takes precedence over basic auth": {
			cfg: config.PulsarConnectionConfig{
				URL:       "pulsar://localhost:6650",
				AuthToken: strPtr("my-jwt-token"),
				BasicAuth: &config.BasicAuth{
					Username: "admin",
					Password: config.NonLoggable("admin-pass"),
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := auth(tc.cfg)
			assert.NoError(t, err)
			if tc.wantNil {
				var nilAuth apacheplsr.Authentication
				assert.Equal(t, nilAuth, got)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}
