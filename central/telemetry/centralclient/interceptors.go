package centralclient

import (
	"strings"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/clientprofile"
	"github.com/stackrox/rox/pkg/glob"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

const userAgentHeaderKey = "User-Agent"

var (
	ignoredPaths = glob.Pattern("{/v1/ping,/v1.PingService/Ping,/v1/metadata,/static/*}")

	permanentTelemetryCampaign = clientprofile.RuleSet{
		{
			Headers: clientprofile.GlobMap{
				userAgentHeaderKey:                  "*roxctl*",
				clientconn.RoxctlCommandHeader:      clientprofile.NoHeaderOrAnyValue,
				clientconn.RoxctlCommandIndexHeader: clientprofile.NoHeaderOrAnyValue,
				clientconn.ExecutionEnvironment:     clientprofile.NoHeaderOrAnyValue,
			},
		},
		{
			Path: glob.Pattern("/v1/clusters").Ptr(),
			Headers: clientprofile.GlobMap{
				// ServiceNow default User-Agent includes "ServiceNow", but
				// customers are free to change it.
				// See https://support.servicenow.com/kb?id=kb_article_view&sysparm_article=KB1511513.
				userAgentHeaderKey: "*ServiceNow*",
				"Rh-*":             clientprofile.NoHeaderOrAnyValue,
			},
		},
		// Capture requests from GitHub action user agents.
		// See https://github.com/stackrox/central-login/blob/68785c129f3faba128d820cfe767558287be53a3/src/main.ts#L73
		// and https://github.com/stackrox/roxctl-installer-action/blob/47fb4f5b275066b8322369e6e33fa010915b0d13/action.yml#L59.
		clientprofile.HeaderPattern(userAgentHeaderKey, "*-GHA*"),
		{
			// Capture SBOM generation requests. Corresponding handler in central/image/service/http_handler.go.
			Path:    glob.Pattern("/api/v1/images/sbom").Ptr(),
			Method:  glob.Pattern("POST").Ptr(),
			Headers: clientprofile.GlobMap{userAgentHeaderKey: clientprofile.NoHeaderOrAnyValue},
		},
		// Capture Jenkins Plugin requests
		clientprofile.HeaderPattern(userAgentHeaderKey, "*stackrox-container-image-scanner*"),
		apiPathsCampaign(),
		userAgentsCampaign(),
	}
)

// apiPathsCampaign constructs a rule from the apiWhiteList setting when that
// setting is non-empty.
// The rule matches requests whose path fits the provided glob pattern and
// requires only the presence (or any value) of a User-Agent header.
// Returns nil when the apiWhiteList setting is empty.
func apiPathsCampaign() *clientprofile.Rule {
	if pattern := apiWhiteList.Setting(); pattern != "" {
		return &clientprofile.Rule{
			Path: glob.Pattern("{" + pattern + "}").Ptr(),
			Headers: clientprofile.GlobMap{
				userAgentHeaderKey: clientprofile.NoHeaderOrAnyValue,
			},
		}
	}
	return nil
}

// userAgentsCampaign constructs a rule that matches the "User-Agent" header
// against the glob pattern defined by the userAgentsList setting.
// If the setting is empty, it returns nil.
func userAgentsCampaign() *clientprofile.Rule {
	if pattern := userAgentsList.Setting(); pattern != "" {
		return clientprofile.HeaderPattern(userAgentHeaderKey, glob.Pattern("{"+pattern+"}"))
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
		return !ignoredPaths.Match(rp.Path) && c.telemetryCampaign.CountMatched(rp,
			func(_ *clientprofile.Rule, h clientprofile.Headers) {
				for k, values := range h {
					props[k] = strings.Join(values, "; ")
				}
			}) > 0
	}
}
