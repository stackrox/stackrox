package service

import (
	"testing"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovedHTTPScopes(t *testing.T) {
	for name, tc := range map[string]struct {
		permissionSet string
		requested     []*apiV1.HTTPRequestScope
		expected      tokens.HTTPRequestScopes
		invalid       bool
	}{
		"console permission set without requested HTTP scopes": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			expected:      centralproxy.ConsoleHTTPScopes,
		},
		"console subset": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/api/graphql", Methods: []string{"POST"}}},
			expected:      tokens.HTTPRequestScopes{"/api/graphql": {"POST"}},
		},
		"console deny-all subset": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/api/graphql"}},
			expected:      tokens.HTTPRequestScopes{"/api/graphql": {}},
		},
		"future metrics profile": {
			permissionSet: centralproxy.MetricsPermissionSetID,
			expected:      centralproxy.MetricsHTTPScopes,
		},
		"metrics cannot ask for console": {
			permissionSet: centralproxy.MetricsPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/metadata", Methods: []string{"GET"}}},
			invalid:       true,
		},
		"metrics cannot ask for POST": {
			permissionSet: centralproxy.MetricsPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/metrics", Methods: []string{"POST"}}},
			invalid:       true,
		},
		"duplicate path": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			requested: []*apiV1.HTTPRequestScope{
				{Path: "/api/graphql", Methods: []string{"POST"}},
				{Path: "/api/graphql"},
			},
			invalid: true,
		},
		"wildcard path": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/*", Methods: []string{"GET"}}},
			invalid:       true,
		},
		"unsupported set": {permissionSet: "unknown", invalid: true},
	} {
		t.Run(name, func(t *testing.T) {
			actual, err := approvedHTTPScopes(&v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetIds:  []string{tc.permissionSet},
				HttpRequestScopes: tc.requested,
			})
			if tc.invalid {
				assert.ErrorIs(t, err, errox.InvalidArgs)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
