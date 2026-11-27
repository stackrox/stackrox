package centralproxy

import "github.com/stackrox/rox/pkg/auth/tokens"

// ConsoleHTTPScopes contains the console plugin's default HTTP routes and methods.
var ConsoleHTTPScopes = tokens.HTTPRequestScopes{
	"/api/graphql":        {"POST"},
	"/static/ocp-plugin/": {"GET"},
	"/v1/config/public":   {"GET"},
	"/v1/deployments":     {"GET"},
	"/v1/featureflags":    {"GET"},
	"/v1/metadata":        {"GET"},
	"/v1/mypermissions":   {"GET"},
	"/v1/telemetry/":      {"GET"},
}

// MetricsHTTPScopes confines the default metrics client to GET /metrics.
var MetricsHTTPScopes = tokens.HTTPRequestScopes{"/metrics": {"GET"}}
