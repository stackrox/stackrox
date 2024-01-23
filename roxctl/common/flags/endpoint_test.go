package flags

import (
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointAndPlaintextSetting(t *testing.T) {
	truth := false
	endpointChanged = &truth
	plaintextSet = &truth

	testCases := []struct {
		endpoint     string
		host         string
		usePlaintext bool
		err          string
	}{
		{
			endpoint: "localhost:8443",
			host:     "localhost:8443",
		},
		{
			endpoint: "https://localhost",
			host:     "localhost:443",
		},
		{
			endpoint: "https://example.com:443",
			host:     "example.com:443",
		},
		{
			endpoint:     "http://example.com:80",
			host:         "example.com:80",
			usePlaintext: true,
		},
		{
			endpoint: "https://example.com",
			host:     "example.com:443",
		},
		{
			endpoint:     "http://example.com",
			host:         "example.com:80",
			usePlaintext: true,
		},
		{
			endpoint:     "http://128.66.0.1",
			host:         "128.66.0.1:80",
			usePlaintext: true,
		},
		{
			endpoint: "128.66.0.1",
			err:      "invalid endpoint: address 128.66.0.1: missing port in address, the scheme should be: http(s)://<endpoint>:<port>",
		},
		{
			endpoint: "example.com",
			err:      "invalid endpoint: address example.com: missing port in address, the scheme should be: http(s)://<endpoint>:<port>",
		},
		{
			endpoint: "example.com:80:80",
			err:      "invalid endpoint: address example.com:80:80: too many colons in address, the scheme should be: http(s)://<endpoint>:<port>",
		},
		{
			endpoint: "host:port",
			host:     "host:port",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.endpoint, func(t *testing.T) {
			t.Setenv(env.EndpointEnv.EnvVar(), tc.endpoint)
			host, usePlaintext, err := EndpointAndPlaintextSetting()
			if tc.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
			assert.Equal(t, tc.host, host)
			assert.Equal(t, tc.usePlaintext, usePlaintext)
		})
	}
}
