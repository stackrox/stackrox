package centralproxy

import (
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/set"
)

// AllowedProxyPaths is the path-only projection of the supported proxy token
// profiles. Central advertises it to Sensor via the CentralHello handshake.
//
// Matching semantics (enforced by the allowedpaths package on the Sensor side):
//   - Entries ending with "/" are treated as prefixes: any request path
//     starting with that prefix is allowed.
//   - Entries without a trailing "/" require an exact match.
//
// IMPORTANT: The proxy buffers request bodies in memory for retry support.
// When adding new paths, ensure the endpoints do not expect request bodies
// exceeding the buffer limit.
var AllowedProxyPaths = allowedPathsForProfiles(ConsoleHTTPScopes, MetricsHTTPScopes)

func allowedPathsForProfiles(profiles ...tokens.HTTPRequestScopes) set.FrozenStringSet {
	var paths []string
	for _, profile := range profiles {
		for path := range profile {
			paths = append(paths, path)
		}
	}
	return set.NewFrozenStringSet(paths...)
}
