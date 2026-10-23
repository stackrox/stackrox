package centralproxy

import "github.com/stackrox/rox/pkg/auth/tokens"

// ConsoleHTTPScopes contains the methods and routes needed by the OCP plugin.
// Paths ending in a slash include descendants; other paths match exactly.
var ConsoleHTTPScopes = []tokens.HTTPRequestScope{
	{Method: "POST", Path: "/api/graphql"},
	{Method: "GET", Path: "/static/ocp-plugin/"},
	{Method: "GET", Path: "/v1/config/public"},
	{Method: "GET", Path: "/v1/deployments"},
	{Method: "GET", Path: "/v1/featureflags"},
	{Method: "GET", Path: "/v1/metadata"},
	{Method: "GET", Path: "/v1/mypermissions"},
	{Method: "GET", Path: "/v1/telemetry/"},
}

// MetricsHTTPScopes confines Sensor-issued metrics tokens to the metrics endpoint.
var MetricsHTTPScopes = []tokens.HTTPRequestScope{{Method: "GET", Path: "/metrics"}}
