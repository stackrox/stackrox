package conf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const portToAdd = 1234

func TestAddressPort(t *testing.T) {
	testCases := []struct {
		addr      string
		withPort  string
		expectErr bool
	}{
		{
			addr:     "localhost:8443",
			withPort: "localhost:8443",
		},
		{
			addr:     "1.1.1.1",
			withPort: "1.1.1.1:1234",
		},
		{
			addr:      "::1:80",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.addr, func(t *testing.T) {
			withPort, err := ensureHasPort(tc.addr, portToAdd)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.withPort, withPort)
			}
		})
	}
}

func TestCentralEndpointDefaulting(t *testing.T) {
	testCases := []struct {
		provided string
		expected string
	}{
		{
			provided: "http://localhost:8080",
			expected: "localhost:8080",
		},
		{
			provided: "http://localhost",
			expected: "localhost",
		},
		{
			provided: "https://localhost:8443",
			expected: "localhost:8443",
		},
		{
			provided: "https://localhost",
			expected: "localhost:443",
		},
		{
			provided: "192.168.1.0",
			expected: "192.168.1.0",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.provided, func(t *testing.T) {
			initCentralEndpoint(tc.provided)
			assert.Equal(t, tc.expected, CentralEndpoint)
		})
	}
}
