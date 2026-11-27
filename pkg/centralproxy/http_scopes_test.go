package centralproxy

import (
	"maps"
	"slices"
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultProxyScopes(t *testing.T) {
	require.NoError(t, tokens.ValidateHTTPRequestScopes(ConsoleHTTPScopes))
	require.NoError(t, tokens.ValidateHTTPRequestScopes(MetricsHTTPScopes))
	assert.ElementsMatch(t, slices.Collect(maps.Keys(ConsoleHTTPScopes)), AllowedProxyPaths.AsSlice())
	assert.False(t, AllowedProxyPaths.Contains("/metrics"), "legacy Sensors only receive console routes")
}
