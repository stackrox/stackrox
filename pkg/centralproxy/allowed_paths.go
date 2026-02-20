package centralproxy

import "github.com/stackrox/rox/pkg/set"

// AllowedProxyPaths defines the paths that Sensor's Central proxy is
// permitted to forward. These are communicated to Sensor via the
// CentralHello handshake.
//
// Matching semantics (enforced by the allowedpaths package on the Sensor side):
//   - Entries ending with "/" are treated as prefixes: any request path
//     starting with that prefix is allowed.
//   - Entries without a trailing "/" require an exact match.
var AllowedProxyPaths = set.NewFrozenStringSet(
	"/api/graphql",
	"/static/ocp-plugin/",
	"/v1/config/public",
	"/v1/deployments",
	"/v1/featureflags",
	"/v1/metadata",
	"/v1/mypermissions",
	"/v1/telemetry/",
)
