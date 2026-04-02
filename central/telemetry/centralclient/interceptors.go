package centralclient

import (
	"strings"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/glob"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

const (
	// The header is set by the RHACS ServiceNow integration.
	// See https://github.com/stackrox/service-now/blob/9d1df943f5f0b3052df97c6272814e2303f17685/52616ff6938a1a50c52a72856aba10fd/update/sys_script_include_2b362bbe938a1a50c52a72856aba10b3.xml#L80.
	snowIntegrationHeader = "Rh-Servicenow-Integration"

	userAgentHeaderKey = "User-Agent"
)

var (
	ignoredPaths = glob.Pattern("{/v1/ping,/v1.PingService/Ping,/v1/metadata,/static/*}")

	permanentTelemetryCampaign = phonehome.APICallCampaign{
		{
			Headers: phonehome.GlobMap{
				userAgentHeaderKey:                  "*roxctl*",
				clientconn.RoxctlCommandHeader:      phonehome.NoHeaderOrAnyValue,
				clientconn.RoxctlCommandIndexHeader: phonehome.NoHeaderOrAnyValue,
				clientconn.ExecutionEnvironment:     phonehome.NoHeaderOrAnyValue,
			},
		},
		{
			Path: glob.Pattern("/v1/clusters").Ptr(),
			Headers: phonehome.GlobMap{
				// ServiceNow default User-Agent includes "ServiceNow", but
				// customers are free to change it.
				// See https://support.servicenow.com/kb?id=kb_article_view&sysparm_article=KB1511513.
				userAgentHeaderKey:    "*ServiceNow*",
				snowIntegrationHeader: phonehome.NoHeaderOrAnyValue,
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
			Headers: phonehome.GlobMap{userAgentHeaderKey: phonehome.NoHeaderOrAnyValue},
		},
		// Capture Jenkins Plugin requests
		phonehome.HeaderPattern(userAgentHeaderKey, "*stackrox-container-image-scanner*"),
		apiPathsCampaign(),
		userAgentsCampaign(),
	}
)

// apiPathsCampaign constructs an API paths campaign from the apiWhiteList
// apiPathsCampaign constructs an APICallCampaignCriterion from the apiWhiteList setting when that setting is non-empty.
// The criterion matches requests whose path fits the provided glob pattern and requires only the presence (or any value) of a User-Agent header.
// Returns nil when the apiWhiteList setting is empty.
func apiPathsCampaign() *phonehome.APICallCampaignCriterion {
	if pattern := apiWhiteList.Setting(); pattern != "" {
		return &phonehome.APICallCampaignCriterion{
			Path: glob.Pattern("{" + pattern + "}").Ptr(),
			Headers: phonehome.GlobMap{
				userAgentHeaderKey: phonehome.NoHeaderOrAnyValue,
			},
		}
	}
	return nil
}

// userAgentsCampaign constructs an User-Agent campaign from the userAgentsList
// userAgentsCampaign constructs an APICallCampaignCriterion that matches the "User-Agent" header against the glob pattern defined by the userAgentsList setting.
// If the setting is empty, it returns nil.
func userAgentsCampaign() *phonehome.APICallCampaignCriterion {
	if pattern := userAgentsList.Setting(); pattern != "" {
		return phonehome.HeaderPattern(userAgentHeaderKey, glob.Pattern("{"+pattern+"}"))
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
			func(_ *phonehome.APICallCampaignCriterion, h phonehome.Headers) {
				for k, values := range h {
					props[k] = strings.Join(values, "; ")
				}
			}) > 0
	}
}
