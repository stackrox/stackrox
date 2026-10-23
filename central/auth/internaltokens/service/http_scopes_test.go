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
		"console accepts explicit non-profile route": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET"}}},
			expected:      tokens.HTTPRequestScopes{"/v1/images": {"GET"}},
		},
		"console needs an allowed method": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/api/graphql"}},
			invalid:       true,
		},
		"future metrics profile": {
			permissionSet: centralproxy.MetricsPermissionSetID,
			expected:      centralproxy.MetricsHTTPScopes,
		},
		"metrics accepts explicit non-profile read route": {
			permissionSet: centralproxy.MetricsPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/metadata", Methods: []string{"GET"}}},
			expected:      tokens.HTTPRequestScopes{"/v1/metadata": {"GET"}},
		},
		"metrics accepts explicit POST route": {
			permissionSet: centralproxy.MetricsPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/metrics", Methods: []string{"POST"}}},
			expected:      tokens.HTTPRequestScopes{"/metrics": {"POST"}},
		},
		"duplicate path": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			requested: []*apiV1.HTTPRequestScope{
				{Path: "/api/graphql", Methods: []string{"POST"}},
				{Path: "/api/graphql"},
			},
			invalid: true,
		},
		"escaped path": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/%2A", Methods: []string{"GET"}}},
			invalid:       true,
		},
		"generic set can use non-profile route": {
			permissionSet: "custom-read-only-set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images/", Methods: []string{"GET", "HEAD"}}},
			expected:      tokens.HTTPRequestScopes{"/v1/images/": {"GET", "HEAD"}},
		},
		"generic set needs HTTP scope": {permissionSet: "custom-read-only-set", invalid: true},
		"generic set cannot have only deny rules": {
			permissionSet: "custom-read-only-set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images"}},
			invalid:       true,
		},
		"generic set rejects escaped path": {
			permissionSet: "custom-read-only-set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/%69mages", Methods: []string{"GET"}}},
			invalid:       true,
		},
		"generic set rejects invalid method": {
			permissionSet: "custom-read-only-set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET /"}}},
			invalid:       true,
		},
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
