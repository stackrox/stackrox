package service

import (
	"testing"
	"time"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestEnforcePermissionSetHTTPScopes(t *testing.T) {
	for name, tc := range map[string]struct {
		permissionSet string
		requested     []*apiV1.HTTPRequestScope
		expected      tokens.HTTPRequestScopes
		invalid       bool
	}{
		"console permission set uses built-in scopes when omitted": {
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
		"metrics permission set uses built-in scopes when omitted": {
			permissionSet: centralproxy.MetricsPermissionSetID,
			expected:      centralproxy.MetricsHTTPScopes,
		},
		"metrics rejects non-profile read route": {
			permissionSet: centralproxy.MetricsPermissionSetID,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/metadata", Methods: []string{"GET"}}},
			invalid:       true,
		},
		"metrics rejects POST route": {
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
			ctx := sensorContext(t, gomock.NewController(t), fixtureconsts.Cluster1)
			_, actual, err := defaultTokenPolicy().enforce(ctx, &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetIds:  []string{tc.permissionSet},
				ClusterScopes:     []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1, FullClusterAccess: true}},
				HttpRequestScopes: tc.requested,
				Lifetime:          durationpb.New(time.Minute),
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

func TestEnforceLegacyPermissionsWithoutHTTPScopes(t *testing.T) {
	ctx := sensorContext(t, gomock.NewController(t), fixtureconsts.Cluster1)
	_, scopes, err := defaultTokenPolicy().enforce(ctx, &v1.GenerateTokenForPermissionsAndScopeRequest{
		Permissions: map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
		Lifetime:    durationpb.New(time.Minute),
	})
	require.NoError(t, err)
	assert.Nil(t, scopes)
}
