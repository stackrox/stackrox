package centralproxy

import (
	"maps"
	"slices"

	"github.com/stackrox/rox/pkg/set"
)

// AllowedProxyPaths is the legacy console path compatibility contract advertised
// in CentralHello. Sensors with per-client policy enforce that policy instead.
// Metrics are excluded because older Sensors do not have a metrics proxy client.
//
// Matching semantics (enforced by the allowedpaths package on the Sensor side):
//   - Entries ending with "/" are treated as prefixes: any request path
//     starting with that prefix is allowed.
//   - Entries without a trailing "/" require an exact match.
//
// IMPORTANT: The proxy buffers request bodies in memory for retry support.
// When adding new paths, ensure the endpoints do not expect request bodies
// exceeding the buffer limit.
var AllowedProxyPaths = set.NewFrozenStringSet(slices.Collect(maps.Keys(ConsoleHTTPScopes))...)
