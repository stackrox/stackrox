package centralproxy

import "github.com/stackrox/rox/pkg/auth/tokens"

// ConsoleHTTPScopes contains the methods and routes needed by the OCP plugin.
// Paths ending in a slash include descendants; other paths match exactly.
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

// MetricsHTTPScopes confines Sensor-issued metrics tokens to the metrics endpoint.
var MetricsHTTPScopes = tokens.HTTPRequestScopes{"/metrics": {"GET"}}
