package centralclient

import (
	"strings"

	"github.com/gobwas/glob"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// The header is set by the RHACS ServiceNow integration.
// See https://github.com/stackrox/service-now/blob/9d1df943f5f0b3052df97c6272814e2303f17685/52616ff6938a1a50c52a72856aba10fd/update/sys_script_include_2b362bbe938a1a50c52a72856aba10b3.xml#L80.
const snowIntegrationHeader = "Rh-ServiceNow-Integration"
const userAgentHeaderKey = "User-Agent"

var (
	ignoredPaths = glob.MustCompile("{/v1/ping,/v1.PingService/Ping,/v1/metadata,/static/*}")

	permanentTelemetryCampaign = phonehome.APICallCampaign{
		{
			Headers: map[string]phonehome.Pattern{
				userAgentHeaderKey:                  "*roxctl*",
				clientconn.RoxctlCommandHeader:      phonehome.NoHeaderOrAnyValue,
				clientconn.RoxctlCommandIndexHeader: phonehome.NoHeaderOrAnyValue,
				clientconn.ExecutionEnvironment:     phonehome.NoHeaderOrAnyValue,
			},
		},
		{
			Path: phonehome.Pattern("/v1/clusters").Ptr(),
			Headers: map[string]phonehome.Pattern{
				// ServiceNow default User-Agent includes "ServiceNow", but
				// customers are free to change it.
				// See https://support.servicenow.com/kb?id=kb_article_view&sysparm_article=KB1511513.
				userAgentHeaderKey:    "*ServiceNow*",
				snowIntegrationHeader: phonehome.NoHeaderOrAnyValue,
			},
		},
		{
			// Capture SBOM generation requests. Corresponding handler in central/image/service/http_handler.go.
			Path:    phonehome.Pattern("/api/v1/images/sbom").Ptr(),
			Method:  phonehome.Pattern("POST").Ptr(),
			Headers: map[string]phonehome.Pattern{userAgentHeaderKey: phonehome.NoHeaderOrAnyValue},
		},
		apiPathsCampaign(),
		userAgentsCampaign(),
	}
	campaignMux       sync.RWMutex
	telemetryCampaign phonehome.APICallCampaign

	interceptors = map[string][]phonehome.Interceptor{
		"API Call": {apiCall, addDefaultProps},
	}
)

// apiPathsCampaign constructs an API paths campaign from the apiWhiteList
// environment variable.
func apiPathsCampaign() *phonehome.APICallCampaignCriterion {
	if pattern := apiWhiteList.Setting(); pattern != "" {
		return &phonehome.APICallCampaignCriterion{
			Path: phonehome.Pattern("{" + pattern + "}").Ptr(),
			Headers: map[string]phonehome.Pattern{
				userAgentHeaderKey: phonehome.NoHeaderOrAnyValue,
			},
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
func apiCall(rp *phonehome.RequestParams, props map[string]any) bool {
	campaignMux.RLock()
	defer campaignMux.RUnlock()
	return !ignoredPaths.Match(rp.Path) && telemetryCampaign.CountFulfilled(rp,
		func(cc *phonehome.APICallCampaignCriterion) {
			addCustomHeaders(rp, cc, props)
		}) > 0
}

// addCustomHeaders adds additional properties to the event if the telemetry
// campaign criterion contains a header pattern condition.
func addCustomHeaders(rp *phonehome.RequestParams, cc *phonehome.APICallCampaignCriterion, props map[string]any) {
	if rp.Headers == nil || cc == nil {
		return
	}
	campaignMux.RLock()
	defer campaignMux.RUnlock()
	for header := range cc.Headers {
		values := rp.Headers(header)
		if len(values) != 0 {
			props[header] = strings.Join(values, "; ")
		}
	}
}
