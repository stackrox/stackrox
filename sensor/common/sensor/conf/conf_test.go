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
