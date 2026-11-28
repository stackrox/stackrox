package centralproxy

import (
	"maps"

	"github.com/stackrox/rox/pkg/auth/tokens"
)

// ConsoleDataHTTPScopes contains the console plugin's routes whose responses
// depend on the console permission set: the data routes and the permissions
// response, which reports the token's permission set to the plugin.
var ConsoleDataHTTPScopes = tokens.HTTPRequestScopes{
	"/api/graphql":      {"POST"},
	"/v1/deployments":   {"GET"},
	"/v1/mypermissions": {"GET"},
}

// ConsoleAuthOnlyHTTPScopes contains the console plugin's routes that carry
// no data and are served with authentication-only tokens.
var ConsoleAuthOnlyHTTPScopes = tokens.HTTPRequestScopes{
	"/static/ocp-plugin/": {"GET"},
	"/v1/config/public":   {"GET"},
	"/v1/featureflags":    {"GET"},
	"/v1/metadata":        {"GET"},
	"/v1/telemetry/":      {"GET"},
}

// ConsoleHTTPScopes contains every console plugin route.
var ConsoleHTTPScopes = func() tokens.HTTPRequestScopes {
	scopes := make(tokens.HTTPRequestScopes, len(ConsoleDataHTTPScopes)+len(ConsoleAuthOnlyHTTPScopes))
	maps.Copy(scopes, ConsoleDataHTTPScopes)
	maps.Copy(scopes, ConsoleAuthOnlyHTTPScopes)
	return scopes
}()

// MetricsHTTPScopes confines the default metrics client to GET /metrics.
var MetricsHTTPScopes = tokens.HTTPRequestScopes{"/metrics": {"GET"}}
