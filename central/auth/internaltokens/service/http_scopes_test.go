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

func TestNamedProxyTokenScopes(t *testing.T) {
	for name, tc := range map[string]struct {
		permissionSet string
		requested     []*apiV1.HTTPRequestScope
		expected      tokens.HTTPRequestScopes
		invalid       bool
	}{
		"console scope": {
			permissionSet: centralproxy.ConsoleProxyPermissionSetName,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/api/graphql", Methods: []string{"POST"}}},
			expected:      tokens.HTTPRequestScopes{"/api/graphql": {"POST"}},
		},
		"custom scope outside console profile": {
			permissionSet: "custom read-only set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images/", Methods: []string{"GET"}}},
			expected:      tokens.HTTPRequestScopes{"/v1/images/": {"GET"}},
		},
		"no scope": {permissionSet: "custom read-only set", invalid: true},
		"no method": {
			permissionSet: "custom read-only set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images/"}}, invalid: true,
		},
		"duplicate path": {
			permissionSet: "custom read-only set",
			requested: []*apiV1.HTTPRequestScope{
				{Path: "/v1/images/", Methods: []string{"GET"}}, {Path: "/v1/images/", Methods: []string{"HEAD"}},
			}, invalid: true,
		},
		"invalid path": {
			permissionSet: "custom read-only set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/%6detrics", Methods: []string{"GET"}}}, invalid: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := sensorContext(t, gomock.NewController(t), fixtureconsts.Cluster1)
			_, actual, err := defaultTokenPolicy().enforce(ctx, &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{tc.permissionSet},
				ClusterScopes:      []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1, FullClusterAccess: true}},
				HttpRequestScopes:  tc.requested,
				Lifetime:           durationpb.New(time.Minute),
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
