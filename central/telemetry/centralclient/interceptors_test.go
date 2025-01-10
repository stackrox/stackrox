package centralclient

import (
	"testing"

	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stretchr/testify/assert"
)

func Test_apiCall(t *testing.T) {
	cases := map[string]struct {
		rp       *phonehome.RequestParams
		expected bool
	}{
		"roxctl": {
			rp: &phonehome.RequestParams{
				UserAgent: "Some roxctl client",
				Method:    "GET",
				Path:      "/v1/endpoint",
				Code:      200,
			},
			expected: true,
		},
		"not roxctl": {
			rp: &phonehome.RequestParams{
				UserAgent: "Some other client",
				Method:    "GET",
				Path:      "/v1/endpoint",
				Code:      200,
			},
			expected: false,
		},
		"roxctl ignored path": {
			rp: &phonehome.RequestParams{
				UserAgent: "Some roxctl client",
				Method:    "GET",
				Path:      ignoredPaths[0],
				Code:      200,
			},
			expected: false,
		},
		"ServiceNow clusters": {
			rp: &phonehome.RequestParams{
				UserAgent: "Some ServiceNow client",
				Method:    "GET",
				Path:      "/v1/clusters",
				Code:      200,
			},
			expected: true,
		},
		"ServiceNow deployments": {
			rp: &phonehome.RequestParams{
				UserAgent: "Some ServiceNow client",
				Method:    "GET",
				Path:      "/v1/deployments",
				Code:      200,
			},
			expected: false,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, apiCall(c.rp, nil))
		})
	}
}
