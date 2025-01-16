package centralclient

import (
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withUserAgent(_ *testing.T, headers map[string][]string, ua string) func(string) []string {
	return func(key string) []string {
		if http.CanonicalHeaderKey(key) == userAgentHeaderKey {
			return []string{ua}
		}
		return headers[key]
	}
}

func Test_apiCall(t *testing.T) {
	cases := map[string]struct {
		rp       *phonehome.RequestParams
		expected bool
	}{
		"roxctl": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some roxctl client"),
				Method:  "GET",
				Path:    "/v1/endpoint",
				Code:    200,
			},
			expected: true,
		},
		"not roxctl": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some other client"),
				Method:  "GET",
				Path:    "/v1/endpoint",
				Code:    200,
			},
			expected: false,
		},
		"roxctl ignored path": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some roxctl client"),
				Method:  "GET",
				Path:    "/v1/ping",
				Code:    200,
			},
			expected: false,
		},
		"ServiceNow clusters": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some ServiceNow client"),
				Method:  "GET",
				Path:    "/v1/clusters",
				Code:    200,
			},
			expected: true,
		},
		"ServiceNow deployments": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some ServiceNow client"),
				Method:  "GET",
				Path:    "/v1/deployments",
				Code:    200,
			},
			expected: false,
		},
		"ServiceNow from integration": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, map[string][]string{
					snowIntegrationHeader: {"v1.0.3"},
				}, "RHACS Integration ServiceNow client"),
				Method: "GET",
				Path:   "/v1/clusters",
				Code:   200,
			},
			expected: true,
		},
	}
	require.NoError(t, telemetryCampaign.Compile())
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, apiCall(c.rp, nil))
		})
	}
}

func Test_addCustomHeaders(t *testing.T) {
	t.Run(snowIntegrationHeader, func(t *testing.T) {
		rp := &phonehome.RequestParams{
			Method: "GET",
			Path:   "/v1/clusters",
			Code:   200,
			Headers: func(h string) []string {
				return map[string][]string{
					userAgentHeaderKey:    {"RHACS Integration ServiceNow client"},
					snowIntegrationHeader: {"v1.0.3", "beta"},
				}[h]
			},
		}
		props := map[string]any{}
		addCustomHeaders(rp, props)
		assert.Equal(t, map[string]any{
			userAgentHeaderKey:    "RHACS Integration ServiceNow client",
			snowIntegrationHeader: "v1.0.3; beta",
		}, props)
	})
	t.Run("3rd-party Integration", func(t *testing.T) {
		rp := &phonehome.RequestParams{
			Method: "GET",
			Path:   "/v1/clusters",
			Code:   200,
			Headers: func(h string) []string {
				return map[string][]string{
					userAgentHeaderKey:      {"ServiceNow"},
					"3rd-party-integration": {"v1.0.3", "beta"},
				}[h]
			},
		}
		props := map[string]any{}
		addCustomHeaders(rp, props)
		assert.Equal(t, map[string]any{
			userAgentHeaderKey: "ServiceNow",
		}, props)
	})
	t.Run("roxctl", func(t *testing.T) {
		rp := &phonehome.RequestParams{
			Method: "GET",
			Path:   "/v1/clusters",
			Code:   200,
			Headers: func(h string) []string {
				return map[string][]string{
					userAgentHeaderKey:                  {"roxctl"},
					clientconn.RoxctlCommandHeader:      {"central"},
					clientconn.RoxctlCommandIndexHeader: {"1"},
					clientconn.ExecutionEnvironment:     {"github"},
				}[h]
			},
		}
		props := map[string]any{}
		addCustomHeaders(rp, props)
		assert.Equal(t, map[string]any{
			userAgentHeaderKey:                  "roxctl",
			clientconn.RoxctlCommandHeader:      "central",
			clientconn.RoxctlCommandIndexHeader: "1",
			clientconn.ExecutionEnvironment:     "github",
		}, props)
	})
}
