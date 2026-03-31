package centralclient

import (
	"testing"

	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The header is set by the RHACS ServiceNow integration.
// See https://github.com/stackrox/service-now/blob/9d1df943f5f0b3052df97c6272814e2303f17685/52616ff6938a1a50c52a72856aba10fd/update/sys_script_include_2b362bbe938a1a50c52a72856aba10b3.xml#L80.
const snowIntegrationHeader = "Rh-Servicenow-Integration"

func withUserAgent(ua string) phonehome.Headers {
	return phonehome.Headers{userAgentHeaderKey: {ua}}
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
				Headers: withUserAgent("Some roxctl client"),
				Method:  "GET",
				Path:    "/v1/endpoint",
				Code:    200,
			},
			expected:      true,
			expectedProps: map[string]any{"User-Agent": "Some roxctl client"},
		},
		"not roxctl": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent("Some other client"),
				Method:  "GET",
				Path:    "/v1/endpoint",
				Code:    200,
			},
			expected:      false,
			expectedProps: noProps,
		},
		"don't catch user-agent": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent("Some client"),
				Method:  "GET",
				Path:    "/v1/test-endpoint",
				Code:    200,
			},
			expected:      true,
			expectedProps: noProps,
		},
		"roxctl ignored path": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent("Some roxctl client"),
				Method:  "GET",
				Path:    "/v1/ping",
				Code:    200,
			},
			expected:      false,
			expectedProps: noProps,
		},
		"ServiceNow clusters": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent("Some ServiceNow client"),
				Method:  "GET",
				Path:    "/v1/clusters",
				Code:    200,
			},
			expected:      true,
			expectedProps: map[string]any{"User-Agent": "Some ServiceNow client"},
		},
		"ServiceNow deployments": {
			rp: &phonehome.RequestParams{
				Headers: withUserAgent("Some ServiceNow client"),
				Method:  "GET",
				Path:    "/v1/deployments",
				Code:    200,
			},
			expected:      false,
			expectedProps: noProps,
		},
		"ServiceNow from integration": {
			rp: &phonehome.RequestParams{
				Headers: phonehome.Headers{
					snowIntegrationHeader: {"v1.0.3"},
					"User-Agent":          {"RHACS Integration ServiceNow client"},
				},
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
				Headers: withUserAgent("central-login-GHA"),
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
				Headers: withUserAgent("roxctl-installer-GHA"),
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
				Headers: withUserAgent("Some SBOM client"),
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
	anyTestEndpoint := phonehome.PathPattern("*test*")
	c := newCentralClient("test-id")
	c.appendRuntimeCampaign(phonehome.APICallCampaign{anyTestEndpoint})
	apiCallInterceptor := c.apiCallInterceptor()
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			props := make(map[string]any)
			assert.Equal(t, c.expected, apiCallInterceptor(c.rp, props))
			assert.Equal(t, c.expectedProps, props)
		})
	}
}
