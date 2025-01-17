package centralclient

import (
	"strings"

	"github.com/gobwas/glob"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// The header is set by the RHACS ServiceNow integration.
const snowIntegrationHeader = "Rh-ServiceNow-Integration"
const userAgentHeaderKey = "User-Agent"

var (
	ignoredPaths = glob.MustCompile("{/v1/ping,/v1.PingService/Ping,/v1/metadata,/static/*}")

	telemetryCampaign = phonehome.APICallCampaign{
		{
			Headers: map[string]phonehome.Pattern{
				userAgentHeaderKey:                  "*roxctl*",
				clientconn.RoxctlCommandHeader:      phonehome.NoHeaderOrAnyValue,
				clientconn.RoxctlCommandIndexHeader: phonehome.NoHeaderOrAnyValue,
				clientconn.ExecutionEnvironment:     phonehome.NoHeaderOrAnyValue,
			},
		},
		{
			Paths: phonehome.Pattern("/v1/clusters").Ptr(),
			Headers: map[string]phonehome.Pattern{
				userAgentHeaderKey:    "*ServiceNow*",
				snowIntegrationHeader: phonehome.NoHeaderOrAnyValue,
			},
		},
		{
			Paths: phonehome.Pattern(apiWhiteList.Setting()).Ptr(),
			Headers: map[string]phonehome.Pattern{
				userAgentHeaderKey: phonehome.NoHeaderOrAnyValue,
			},
		},
		apiPathsCampaign(),
		userAgentsCampaign(),
	}

	interceptors = map[string][]phonehome.Interceptor{
		"API Call": {apiCall, addDefaultProps, addCustomHeaders},
	}
)

// apiPathsCampaign constructs an API paths campaign from the apiWhiteList
// environment variable.
func apiPathsCampaign() *phonehome.APICallCampaignCriterion {
	if pattern := apiWhiteList.Setting(); pattern != "" {
		return &phonehome.APICallCampaignCriterion{
			Paths: phonehome.Pattern("{" + pattern + "}").Ptr(),
		}
	}
	return nil
}

// userAgentsCampaign constructs an User-Agent campaign from the userAgentsList
// environment variable.
func userAgentsCampaign() *phonehome.APICallCampaignCriterion {
	if pattern := userAgentsList.Setting(); pattern != "" {
		return &phonehome.APICallCampaignCriterion{
			Headers: map[string]phonehome.Pattern{
				userAgentHeaderKey: phonehome.Pattern("{" + pattern + "}"),
			},
		}
	}
	return nil
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
	return !ignoredPaths.Match(rp.Path) && telemetryCampaign.IsFulfilled(rp)
}

// addCustomHeaders adds additional properties to the event if the telemetry
// campaign contains a header pattern condition.
func addCustomHeaders(rp *phonehome.RequestParams, props map[string]any) bool {
	if rp.Headers == nil {
		return true
	}
	for _, c := range telemetryCampaign {
		if c != nil {
			for header := range c.Headers {
				values := rp.Headers(header)
				if len(values) != 0 {
					props[header] = strings.Join(values, "; ")
				}
			}
		}
	}
	return true
}
