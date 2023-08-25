package centralclient

import (
	"strings"

	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	trackedPaths []string
	ignoredPaths = []string{"/v1/ping", "/v1.PingService/Ping", "/v1/metadata", "/static/*"}

	interceptors = map[string][]phonehome.Interceptor{
		"API Call": {apiCall, addDefaultProps},
		"roxctl":   {roxctl, addDefaultProps},
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
// trackedPaths ("*" value enables all paths) and have no match in the
// ignoredPaths list.
func apiCall(rp *phonehome.RequestParams, _ map[string]any) bool {
	return !rp.HasPathIn(ignoredPaths) && rp.HasPathIn(trackedPaths)
}

// roxctl enables the roxctl event.
func roxctl(rp *phonehome.RequestParams, _ map[string]any) bool {
	return strings.Contains(rp.UserAgent, "roxctl")
}
