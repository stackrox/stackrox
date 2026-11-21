package service

import (
	"testing"
	"time"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestEnforcePermissionSetHTTPScopes(t *testing.T) {
	for name, tc := range map[string]struct {
		permissionSet string
		requested     []*apiV1.HTTPRequestScope
		invalid       bool
	}{
		"console permission set without scopes": {
			permissionSet: accesscontrol.K8sCentralProxyUser,
		},
		"console subset": {
			permissionSet: accesscontrol.K8sCentralProxyUser,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/api/graphql", Methods: []string{"POST"}}},
		},
		"console accepts explicit non-profile route": {
			permissionSet: accesscontrol.K8sCentralProxyUser,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET"}}},
		},
		"no scope":     {permissionSet: "custom read-only set"},
		"empty scopes": {permissionSet: "custom read-only set", requested: []*apiV1.HTTPRequestScope{}},
		"nil scope entry": {
			permissionSet: "custom read-only set",
			requested:     []*apiV1.HTTPRequestScope{nil}, invalid: true,
		},
		"empty method": {
			permissionSet: "custom read-only set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images/", Methods: []string{""}}}, invalid: true,
		},
		"invalid method": {
			permissionSet: "custom read-only set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images/", Methods: []string{"get"}}}, invalid: true,
		},
		"only deny rules": {
			permissionSet: "custom read-only set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images/"}},
		},
		"nil methods deny a specific path": {
			permissionSet: "custom read-only set",
			requested: []*apiV1.HTTPRequestScope{
				{Path: "/v1/images/", Methods: []string{"GET"}},
				{Path: "/v1/images/private"},
			},
		},
		"empty methods deny a specific path": {
			permissionSet: "custom read-only set",
			requested: []*apiV1.HTTPRequestScope{
				{Path: "/v1/images/", Methods: []string{"GET"}},
				{Path: "/v1/images/private", Methods: []string{}},
			},
		},
		"multiple deny rules without an allow": {
			permissionSet: "custom read-only set",
			requested: []*apiV1.HTTPRequestScope{
				{Path: "/v1/images/"},
				{Path: "/v1/images/private", Methods: []string{}},
			},
		},
		"console accepts deny-only scopes": {
			permissionSet: accesscontrol.K8sCentralProxyUser,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/api/graphql"}},
		},
		"metrics permission set without scopes": {
			permissionSet: accesscontrol.OpenShiftPrometheus,
		},
		"metrics reader accepts other canonical read route": {
			permissionSet: accesscontrol.OpenShiftPrometheus,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/metadata", Methods: []string{"GET"}}},
		},
		"metrics reader accepts other canonical method": {
			permissionSet: accesscontrol.OpenShiftPrometheus,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/metrics", Methods: []string{"POST"}}},
		},
		"duplicate path": {
			permissionSet: accesscontrol.K8sCentralProxyUser,
			requested: []*apiV1.HTTPRequestScope{
				{Path: "/api/graphql", Methods: []string{"POST"}},
				{Path: "/api/graphql", Methods: []string{"GET"}},
			},
			invalid: true,
		},
		"escaped path": {
			permissionSet: accesscontrol.K8sCentralProxyUser,
			requested:     []*apiV1.HTTPRequestScope{{Path: "/%2A", Methods: []string{"GET"}}},
			invalid:       true,
		},
		"generic set can use non-profile route": {
			permissionSet: "custom-read-only-set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images/", Methods: []string{"GET", "HEAD"}}},
		},
		"generic set accepts only deny rules": {
			permissionSet: "custom-read-only-set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/images"}},
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
		"generic set rejects noncanonical path": {
			permissionSet: "custom-read-only-set",
			requested:     []*apiV1.HTTPRequestScope{{Path: "/v1/../metrics", Methods: []string{"GET"}}},
			invalid:       true,
		},
		"empty method list denies alongside allowed route": {
			permissionSet: "custom-read-only-set",
			requested: []*apiV1.HTTPRequestScope{
				{Path: "/v1/images/", Methods: []string{"GET"}},
				{Path: "/v1/images/private"},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := sensorContext(t, gomock.NewController(t), fixtureconsts.Cluster1)
			request := &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSets: []*v1.ScopedPermissionSet{{
					Selector:          &v1.ScopedPermissionSet_Name{Name: tc.permissionSet},
					HttpRequestScopes: tc.requested,
				}},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1, FullClusterAccess: true}},
				Lifetime:      durationpb.New(time.Minute),
			}
			actual, err := defaultTokenPolicy().enforce(ctx, request)
			if tc.invalid {
				assert.ErrorIs(t, err, errox.InvalidArgs)
				return
			}
			require.NoError(t, err)
			protoassert.Equal(t, request, actual)
		})
	}
}
