package centralclient

import (
	"strings"

	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	trackedPaths []string
	ignoredPaths = []string{"/v1/ping", "/v1.PingService/Ping", "/v1/metadata", "/static/*"}

	interceptors = map[string][]phonehome.Interceptor{
		"API Call": {apiCall},
		"roxctl":   {roxctl},
	}
)

// apiCall enables API Call events for the API paths specified in the
// trackedPaths ("*" value enables all paths) and have no match in the
// ignoredPaths list.
func apiCall(rp *phonehome.RequestParams, props map[string]any) bool {
	if !rp.HasPathIn(ignoredPaths) && rp.HasPathIn(trackedPaths) {
		props["Path"] = rp.Path
		props["Code"] = rp.Code
		props["User-Agent"] = rp.UserAgent
		props["Method"] = rp.Method
		return true
	}
	return false
}

// roxctl enables the roxctl event.
func roxctl(rp *phonehome.RequestParams, props map[string]any) bool {
	if !strings.Contains(rp.UserAgent, "roxctl") {
		return false
	}
	props["Path"] = rp.Path
	props["Code"] = rp.Code
	props["User-Agent"] = rp.UserAgent
	props["Method"] = rp.Method
	return true
}
