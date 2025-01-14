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
		"ServiceNow from integration": {
			rp: &phonehome.RequestParams{
				UserAgent: "RHACS Integration ServiceNow client",
				Method:    "GET",
				Path:      "/v1/whatever",
				Code:      200,
				Headers: func(h string) []string {
					return map[string][]string{
						"RHACS-Integration": {"v1.0.3"},
					}[h]
				},
			},
			expected: true,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, apiCall(c.rp, nil))
		})
	}
}

func Test_addCustomHeaders(t *testing.T) {
	t.Run("RHACS-Integration", func(t *testing.T) {
		rp := &phonehome.RequestParams{
			UserAgent: "RHACS Integration ServiceNow client",
			Method:    "GET",
			Path:      "/v1/clusters",
			Code:      200,
			Headers: func(h string) []string {
				return map[string][]string{
					"RHACS-Integration": {"v1.0.3", "beta"},
				}[h]
			},
		}
		props := map[string]any{}
		addCustomHeaders(rp, props)
		assert.Len(t, props, 1)
		assert.Equal(t, "v1.0.3; beta", props["RHACS-Integration"])
	})
	t.Run("3rd-party Integration", func(t *testing.T) {
		rp := &phonehome.RequestParams{
			UserAgent: "ServiceNow",
			Method:    "GET",
			Path:      "/v1/clusters",
			Code:      200,
			Headers: func(h string) []string {
				return map[string][]string{
					"3rd-party-integration": {"v1.0.3", "beta"},
				}[h]
			},
		}
		props := map[string]any{}
		addCustomHeaders(rp, props)
		assert.Empty(t, props)
	})
}
