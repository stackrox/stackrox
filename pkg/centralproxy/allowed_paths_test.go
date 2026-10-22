package centralproxy

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
)

func TestAllowedPathsForProfiles(t *testing.T) {
	paths := allowedPathsForProfiles(
		tokens.HTTPRequestScopes{
			"/api/graphql": {"GET", "POST"},
		},
		tokens.HTTPRequestScopes{"/metrics": {"GET"}},
	)
	assert.ElementsMatch(t, []string{"/api/graphql", "/metrics"}, paths.AsSlice())

	var advertisedPaths []string
	for _, profile := range []tokens.HTTPRequestScopes{ConsoleHTTPScopes, MetricsHTTPScopes} {
		for path := range profile {
			advertisedPaths = append(advertisedPaths, path)
		}
	}
	assert.ElementsMatch(t, advertisedPaths, AllowedProxyPaths.AsSlice())
	assert.True(t, AllowedProxyPaths.Contains("/metrics"))
}
