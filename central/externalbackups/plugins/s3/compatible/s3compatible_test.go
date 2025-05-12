package s3compatible

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointValidation(t *testing.T) {
	testCases := map[string]struct {
		endpoint          string
		sanitizedEndpoint string
		shouldError       bool
	}{
		"with https prefix": {
			endpoint:          "https://play.min.io",
			sanitizedEndpoint: "https://play.min.io",
		},
		"with http prefix": {
			endpoint:          "http://play.min.io",
			sanitizedEndpoint: "http://play.min.io",
		},
		"without prefix": {
			endpoint:          "play.min.io",
			sanitizedEndpoint: "https://play.min.io",
		},
		"invalid URL": {
			endpoint:    "play%min.io",
			shouldError: true,
		},
		"with trailing slash": {
			endpoint:          "https://play.min.io/",
			sanitizedEndpoint: "https://play.min.io",
		},
	}

	for caseName, testCase := range testCases {
		t.Run(caseName, func(t *testing.T) {
			result, err := validateEndpoint(testCase.endpoint)
			if testCase.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, testCase.sanitizedEndpoint, result)
		})
	}
}
