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

	var consolePaths []string
	for path := range ConsoleHTTPScopes {
		consolePaths = append(consolePaths, path)
	}
	assert.ElementsMatch(t, consolePaths, AllowedProxyPaths.AsSlice())
	assert.False(t, AllowedProxyPaths.Contains("/metrics"))
}
