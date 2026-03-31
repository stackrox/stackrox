package centralclient

import (
	"strings"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/glob"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

const userAgentHeaderKey = "User-Agent"

var (
	ignoredPaths = glob.Pattern("{/v1/ping,/v1.PingService/Ping,/v1/metadata,/static/*}")

	permanentTelemetryCampaign = phonehome.APICallCampaign{
		{
			Headers: map[glob.Pattern]glob.Pattern{
				userAgentHeaderKey:                  "*roxctl*",
				clientconn.RoxctlCommandHeader:      phonehome.NoHeaderOrAnyValue,
				clientconn.RoxctlCommandIndexHeader: phonehome.NoHeaderOrAnyValue,
				clientconn.ExecutionEnvironment:     phonehome.NoHeaderOrAnyValue,
			},
		},
		{
			Path: glob.Pattern("/v1/clusters").Ptr(),
			Headers: map[glob.Pattern]glob.Pattern{
				// ServiceNow default User-Agent includes "ServiceNow", but
				// customers are free to change it.
				// See https://support.servicenow.com/kb?id=kb_article_view&sysparm_article=KB1511513.
				userAgentHeaderKey: "*ServiceNow*",
				"Rh-*":             phonehome.NoHeaderOrAnyValue,
			},
		},
		// Capture requests from GitHub action user agents.
		// See https://github.com/stackrox/central-login/blob/68785c129f3faba128d820cfe767558287be53a3/src/main.ts#L73
		// and https://github.com/stackrox/roxctl-installer-action/blob/47fb4f5b275066b8322369e6e33fa010915b0d13/action.yml#L59.
		phonehome.HeaderPattern(userAgentHeaderKey, "*-GHA*"),
		{
			// Capture SBOM generation requests. Corresponding handler in central/image/service/http_handler.go.
			Path:    glob.Pattern("/api/v1/images/sbom").Ptr(),
			Method:  glob.Pattern("POST").Ptr(),
			Headers: map[glob.Pattern]glob.Pattern{userAgentHeaderKey: phonehome.NoHeaderOrAnyValue},
		},
		// Capture Jenkins Plugin requests
		phonehome.HeaderPattern(userAgentHeaderKey, "*stackrox-container-image-scanner*"),
		apiPathsCampaign(),
		userAgentsCampaign(),
	}
)

// apiPathsCampaign constructs an API paths campaign from the apiWhiteList
// environment variable.
func apiPathsCampaign() *phonehome.APICallCampaignCriterion {
	if pattern := apiWhiteList.Setting(); pattern != "" {
		return &phonehome.APICallCampaignCriterion{
			Path: glob.Pattern("{" + pattern + "}").Ptr(),
			Headers: map[glob.Pattern]glob.Pattern{
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
		return phonehome.HeaderPattern(userAgentHeaderKey, "{"+pattern+"}")
	}
	return nil
}

func addDefaultProps(rp *phonehome.RequestParams, props map[string]any) bool {
	props["Path"] = rp.Path
	props["Code"] = rp.Code
	props["Method"] = rp.Method
	return true
}

// apiCallInterceptor enables API Call events for the API paths specified in the
// trackedPaths ("*" value enables all paths) or for the calls with the
// User-Agent containing the substrings specified in the trackedUserAgents, and
// have no match in the ignoredPaths list.
func (c *CentralClient) apiCallInterceptor() phonehome.Interceptor {
	return func(rp *phonehome.RequestParams, props map[string]any) bool {
		c.campaignMux.RLock()
		defer c.campaignMux.RUnlock()
		return !ignoredPaths.Match(rp.Path) && c.telemetryCampaign.CountFulfilled(rp,
			func(cc *phonehome.APICallCampaignCriterion) {
				addCustomHeaders(rp, cc, props)
			}) > 0
	}
}

// addCustomHeaders adds additional properties to the event if the telemetry
// campaign criterion contains a header pattern condition.
func addCustomHeaders(rp *phonehome.RequestParams, cc *phonehome.APICallCampaignCriterion, props map[string]any) {
	if cc == nil {
		return
	}
	for header := range cc.Headers {
		values, err := rp.Headers.GetAll(header, "*")
		if err != nil {
			return
		}
		for h, v := range values {
			props[h] = strings.Join(v, "; ")
		}
	}
}
