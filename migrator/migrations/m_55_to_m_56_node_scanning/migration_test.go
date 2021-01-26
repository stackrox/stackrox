package m55tom56

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGRPCEndpoint(t *testing.T) {
	cases := []struct {
		httpsEndpoint        string
		expectedGRPCEndpoint string
	}{
		{
			httpsEndpoint:        "https://scanner.stackrox:8080",
			expectedGRPCEndpoint: "scanner.stackrox:8443",
		},
		{
			httpsEndpoint:        "https://scanner-endpoint",
			expectedGRPCEndpoint: "scanner-endpoint:8443",
		},
		{
			httpsEndpoint:        "scanner.stackrox:8080",
			expectedGRPCEndpoint: "scanner.stackrox:8443",
		},
	}
	for _, c := range cases {
		t.Run(c.httpsEndpoint, func(t *testing.T) {
			assert.Equal(t, c.expectedGRPCEndpoint, httpsEndpointToGRPC(c.httpsEndpoint))
		})
	}
}
