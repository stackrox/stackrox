package centralclient

import (
	"strings"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// The header is set by the RHACS ServiceNow integration.
const snowIntegrationHeader = "Rh-ServiceNow-Integration"
const userAgentHeaderKey = "User-Agent"

var (
	ignoredPaths = []string{"/v1/ping", "/v1.PingService/Ping", "/v1/metadata", "/static/*"}

	telemetryCampaign = append(phonehome.APICallCampaign{
		{
			Headers: map[string]string{
				userAgentHeaderKey:                  "*roxctl*",
				clientconn.RoxctlCommandHeader:      phonehome.NoHeaderOrAnyValuePattern,
				clientconn.RoxctlCommandIndexHeader: phonehome.NoHeaderOrAnyValuePattern,
				clientconn.ExecutionEnvironment:     phonehome.NoHeaderOrAnyValuePattern,
			},
		},
		{
			Paths: []string{"/v1/clusters"},
			Headers: map[string]string{
				userAgentHeaderKey:    "*ServiceNow*",
				snowIntegrationHeader: phonehome.NoHeaderOrAnyValuePattern,
			},
		},
		{
			Paths: splitString(apiWhiteList.Setting(), ','),
			Headers: map[string]string{
				userAgentHeaderKey: phonehome.NoHeaderOrAnyValuePattern,
			},
		},
	}, userAgentsCampaigns()...)

	interceptors = map[string][]phonehome.Interceptor{
		"API Call": {apiCall, addDefaultProps, addCustomHeaders},
	}
)

// splitString splits the string s by the sep separator, returning an empty
// slice for empty input string.
func splitString(s string, sep rune) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == sep })
}

func userAgentsCampaigns() []phonehome.APICallCampaignCriterion {
	result := []phonehome.APICallCampaignCriterion{}
	for _, ua := range splitString(userAgentsList.Setting(), ',') {
		result = append(result, phonehome.APICallCampaignCriterion{
			Headers: map[string]string{
				userAgentHeaderKey: ua,
			},
		})
	}
	return result
}

func addDefaultProps(rp *phonehome.RequestParams, props map[string]any) bool {
	props["Path"] = rp.Path
	props["Code"] = rp.Code
	props["Method"] = rp.Method
	return true
}

// apiCall enables API Call events for the API paths specified in the
// trackedPaths ("*" value enables all paths) or for the calls with the
// User-Agent containing the substrings specified in the trackedUserAgents, and
// have no match in the ignoredPaths list.
func apiCall(rp *phonehome.RequestParams, _ map[string]any) bool {
	return !rp.HasPathIn(ignoredPaths) && telemetryCampaign.IsFulfilled(rp)
}

// addCustomHeaders adds additional properties to the event if the telemetry
// campaign contains a header pattern condition.
func addCustomHeaders(rp *phonehome.RequestParams, props map[string]any) bool {
	if rp.Headers == nil {
		return true
	}
	for _, c := range telemetryCampaign {
		for header := range c.Headers {
			values := rp.Headers(header)
			if len(values) != 0 {
				props[header] = strings.Join(values, "; ")
			}
		}
	}
	return true
}
