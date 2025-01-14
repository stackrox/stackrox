package centralclient

import (
	"strings"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

const snowIntegrationHeader = "Rh-ServiceNow-Integration"
const anyValueOrMissing = ""

var (
	ignoredPaths = []string{"/v1/ping", "/v1.PingService/Ping", "/v1/metadata", "/static/*"}

	telemetryCampaign = phonehome.APICallCampaign{
		{
			UserAgents: []string{"roxctl"},
			HeaderPatterns: map[string]string{
				clientconn.RoxctlCommandHeader:      anyValueOrMissing,
				clientconn.RoxctlCommandIndexHeader: anyValueOrMissing,
				clientconn.ExecutionEnvironment:     anyValueOrMissing,
			},
		},
		{
			UserAgents:   []string{"ServiceNow"},
			PathPatterns: []string{"/v1/clusters"},
			HeaderPatterns: map[string]string{
				snowIntegrationHeader: anyValueOrMissing,
			},
		},
		{PathPatterns: strings.FieldsFunc(apiWhiteList.Setting(),
			func(r rune) bool { return r == ',' })},
		{UserAgents: strings.FieldsFunc(userAgentsList.Setting(),
			func(r rune) bool { return r == ',' })},
	}

	interceptors = map[string][]phonehome.Interceptor{
		"API Call": {apiCall, addDefaultProps, addCustomHeaders},
	}
)

func addDefaultProps(rp *phonehome.RequestParams, props map[string]any) bool {
	props["Path"] = rp.Path
	props["Code"] = rp.Code
	props["Method"] = rp.Method
	props["User-Agent"] = rp.UserAgent
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
	for _, c := range telemetryCampaign {
		for header := range c.HeaderPatterns {
			values := rp.Headers(header)
			if len(values) != 0 {
				props[header] = strings.Join(values, "; ")
			}
		}
	}
	return true
}
