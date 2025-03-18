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
	noProps := make(map[string]any)
	cases := map[string]struct {
		rp            *phonehome.RequestParams
		expected      bool
		expectedProps map[string]any
	}{
		"roxctl": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some roxctl client"),
				Method:  "GET",
				Path:    "/v1/endpoint",
				Code:    200,
			},
			expected:      true,
			expectedProps: map[string]any{"User-Agent": "Some roxctl client"},
		},
		"not roxctl": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some other client"),
				Method:  "GET",
				Path:    "/v1/endpoint",
				Code:    200,
			},
			expected:      false,
			expectedProps: noProps,
		},
		"don't catch user-agent": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some client"),
				Method:  "GET",
				Path:    "/v1/test-endpoint",
				Code:    200,
			},
			expected:      true,
			expectedProps: noProps,
		},
		"roxctl ignored path": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some roxctl client"),
				Method:  "GET",
				Path:    "/v1/ping",
				Code:    200,
			},
			expected:      false,
			expectedProps: noProps,
		},
		"ServiceNow clusters": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some ServiceNow client"),
				Method:  "GET",
				Path:    "/v1/clusters",
				Code:    200,
			},
			expected:      true,
			expectedProps: map[string]any{"User-Agent": "Some ServiceNow client"},
		},
		"ServiceNow deployments": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some ServiceNow client"),
				Method:  "GET",
				Path:    "/v1/deployments",
				Code:    200,
			},
			expected:      false,
			expectedProps: noProps,
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
			expectedProps: map[string]any{
				snowIntegrationHeader: "v1.0.3",
				"User-Agent":          "RHACS Integration ServiceNow client",
			},
		},
		"central-login GitHub action": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "central-login-GHA"),
				Method:  "POST",
				Path:    "/v1/auth/m2m/exchange",
			},
			expected: true,
			expectedProps: map[string]any{
				"User-Agent": "central-login-GHA",
			},
		},
		"roxctl-installer GitHub action": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "roxctl-installer-GHA"),
				Method:  "GET",
				Path:    "/api/cli/download/roxctl-linux-amd64",
			},
			expected: true,
			expectedProps: map[string]any{
				"User-Agent": "roxctl-installer-GHA",
			},
		},
		"SBOM generation": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent(t, nil, "Some SBOM client"),
				Method:  "POST",
				Path:    "/api/v1/images/sbom",
				Code:    200,
			},
			expected: true,
			expectedProps: map[string]any{
				"User-Agent": "Some SBOM client",
			},
		},
	}
	require.NoError(t, permanentTelemetryCampaign.Compile())
	anyTestEndpoint := &phonehome.APICallCampaignCriterion{
		Path: phonehome.Pattern("*test*").Ptr(),
	}
	appendRuntimeCampaign(&phonehome.RuntimeConfig{
		APICallCampaign: phonehome.APICallCampaign{anyTestEndpoint},
	})
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			props := make(map[string]any)
			assert.Equal(t, c.expected, apiCall(c.rp, props))
			assert.Equal(t, c.expectedProps, props)
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
		addCustomHeaders(rp, telemetryCampaign[2], props)
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
		addCustomHeaders(rp, telemetryCampaign[1], props)
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
		addCustomHeaders(rp, telemetryCampaign[0], props)
		assert.Equal(t, map[string]any{
			userAgentHeaderKey:                  "roxctl",
			clientconn.RoxctlCommandHeader:      "central",
			clientconn.RoxctlCommandIndexHeader: "1",
			clientconn.ExecutionEnvironment:     "github",
		}, props)
	})
	t.Run("add header from the single criterion", func(t *testing.T) {
		previousCampaign := telemetryCampaign
		telemetryCampaign = append(telemetryCampaign, &phonehome.APICallCampaignCriterion{
			Headers: map[string]phonehome.Pattern{"Custom-Header": ""},
		})
		require.NoError(t, telemetryCampaign.Compile())
		rp := &phonehome.RequestParams{
			Method: "GET",
			Path:   "/v1/config",
			Code:   200,
			Headers: func(h string) []string {
				return map[string][]string{
					userAgentHeaderKey: {"roxctl"},
				}[h]
			},
		}
		props := map[string]any{}
		addCustomHeaders(rp, telemetryCampaign[0], props)
		assert.Equal(t, map[string]any{
			userAgentHeaderKey: "roxctl",
		}, props)
		telemetryCampaign = previousCampaign
	})
}
