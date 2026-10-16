package centralproxy

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
)

func TestAllowedPathsForProfiles(t *testing.T) {
	paths := allowedPathsForProfiles(
		[]tokens.HTTPRequestScope{
			{Path: "/api/graphql", Method: "GET"},
			{Path: "/api/graphql", Method: "POST"},
		},
		[]tokens.HTTPRequestScope{{Path: "/metrics", Method: "GET"}},
	)
	assert.ElementsMatch(t, []string{"/api/graphql", "/metrics"}, paths.AsSlice())

	var consolePaths []string
	for _, scope := range ConsoleHTTPScopes {
		consolePaths = append(consolePaths, scope.Path)
	}
	assert.ElementsMatch(t, consolePaths, AllowedProxyPaths.AsSlice())
	assert.True(t, AllowedProxyPaths.Contains("/metrics"))
}
