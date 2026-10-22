package service

import (
	"testing"
	"time"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authn"
	authnMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
)

// This test is a sanity check in case defaultAllowedPermissions is modified.
func TestDefaultTokenPolicyIsCompatibleWithOCPPlugin(t *testing.T) {
	policy := defaultTokenPolicy()
	require.NotNil(t, policy)

	// Needed for vulnerability management in the ocp console plugin.
	minimumPermissionSet := map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
		"Image":      v1.Access_READ_ACCESS,
	}
	assert.Equal(t, minimumPermissionSet, policy.allowedPermissions,
		"default allowed permissions must be compatible with ocp plugin")
}

func TestValidatePermissions(t *testing.T) {
	policy := newTokenPolicy(0, map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
		"Image":      v1.Access_READ_ACCESS,
	})

	for name, tc := range map[string]struct {
		requested   map[string]v1.Access
		expectError bool
	}{
		"nil permissions": {
			requested: nil,
		},
		"empty permissions": {
			requested: map[string]v1.Access{},
		},
		"valid subset - single": {
			requested: map[string]v1.Access{
				"Deployment": v1.Access_READ_ACCESS,
			},
		},
		"valid subset - both": {
			requested: map[string]v1.Access{
				"Deployment": v1.Access_READ_ACCESS,
				"Image":      v1.Access_READ_ACCESS,
			},
		},
		"lower access than allowed": {
			requested: map[string]v1.Access{
				"Deployment": v1.Access_NO_ACCESS,
			},
		},
		"resource not in allowlist": {
			requested: map[string]v1.Access{
				"NetworkGraph": v1.Access_READ_ACCESS,
			},
			expectError: true,
		},
		"access exceeds allowlist": {
			requested: map[string]v1.Access{
				"Deployment": v1.Access_READ_WRITE_ACCESS,
			},
			expectError: true,
		},
		"mixed - one valid, one not allowed resource": {
			requested: map[string]v1.Access{
				"Deployment":   v1.Access_READ_ACCESS,
				"NetworkGraph": v1.Access_READ_ACCESS,
			},
			expectError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := policy.validatePermissions(tc.requested)
			if tc.expectError {
				assert.ErrorIs(t, err, errox.InvalidArgs)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateClusterScope(t *testing.T) {
	policy := newTokenPolicy(0, nil)

	for name, tc := range map[string]struct {
		scopes          []*v1.ClusterScope
		sensorClusterID string
		expectError     bool
	}{
		"nil scopes": {
			scopes:          nil,
			sensorClusterID: fixtureconsts.Cluster1,
		},
		"empty scopes": {
			scopes:          []*v1.ClusterScope{},
			sensorClusterID: fixtureconsts.Cluster1,
		},
		"matching cluster": {
			scopes: []*v1.ClusterScope{
				{ClusterId: fixtureconsts.Cluster1},
			},
			sensorClusterID: fixtureconsts.Cluster1,
		},
		"multiple matching clusters": {
			scopes: []*v1.ClusterScope{
				{ClusterId: fixtureconsts.Cluster1},
				{ClusterId: fixtureconsts.Cluster1, Namespaces: []string{"ns1"}},
			},
			sensorClusterID: fixtureconsts.Cluster1,
		},
		"mismatched cluster": {
			scopes: []*v1.ClusterScope{
				{ClusterId: fixtureconsts.Cluster2},
			},
			sensorClusterID: fixtureconsts.Cluster1,
			expectError:     true,
		},
		"one matching, one mismatched": {
			scopes: []*v1.ClusterScope{
				{ClusterId: fixtureconsts.Cluster1},
				{ClusterId: fixtureconsts.Cluster2},
			},
			sensorClusterID: fixtureconsts.Cluster1,
			expectError:     true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := policy.validateClusterScope(tc.scopes, tc.sensorClusterID)
			if tc.expectError {
				assert.ErrorIs(t, err, errox.InvalidArgs)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnforceScopedPermissionSets(t *testing.T) {
	policy := defaultTokenPolicy()
	valid := func() *v1.GenerateTokenForPermissionsAndScopeRequest {
		return &v1.GenerateTokenForPermissionsAndScopeRequest{
			ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1, Namespaces: []string{"team-a"}}},
			Lifetime:      durationpb.New(5 * time.Minute),
			ScopedPermissionSets: []*v1.ScopedPermissionSet{
				{PermissionSetName: "Deployment Reader", HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}}},
				{PermissionSetName: "Image Reader", HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET"}}}},
			},
		}
	}
	for name, change := range map[string]func(*v1.GenerateTokenForPermissionsAndScopeRequest){
		"valid": func(*v1.GenerateTokenForPermissionsAndScopeRequest) {},
		"overlapping route": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.ScopedPermissionSets[1].HttpRequestScopes[0].Path = "/v1/deployments"
		},
		"duplicate permission set": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.ScopedPermissionSets[1].PermissionSetName = "Deployment Reader"
		},
		"missing methods": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.ScopedPermissionSets[0].HttpRequestScopes[0].Methods = nil
		},
		"missing all scopes": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			for _, grant := range r.ScopedPermissionSets {
				grant.HttpRequestScopes = nil
			}
		},
		"same path with distinct methods": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.ScopedPermissionSets[1].HttpRequestScopes[0].Path = "/v1/deployments"
			r.ScopedPermissionSets[1].HttpRequestScopes[0].Methods = []string{"POST"}
		},
		"global legacy scope": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.HttpRequestScopes = []*apiV1.HTTPRequestScope{{Path: "/", Methods: []string{"GET"}}}
		},
		"legacy permissions": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.Permissions = map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS}
		},
		"other cluster": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.ClusterScopes[0].ClusterId = fixtureconsts.Cluster2
		},
		"no cluster": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) { r.ClusterScopes = nil },
		"metrics combined": func(r *v1.GenerateTokenForPermissionsAndScopeRequest) {
			r.ScopedPermissionSets[1].PermissionSetName = centralproxy.MetricsPermissionSetName
		},
	} {
		t.Run(name, func(t *testing.T) {
			req := valid()
			change(req)
			_, scopes, err := policy.enforce(sensorContext(t, gomock.NewController(t), fixtureconsts.Cluster1), req)
			if name == "valid" || name == "same path with distinct methods" {
				require.NoError(t, err)
				assert.Nil(t, scopes)
			} else {
				require.ErrorIs(t, err, errox.InvalidArgs)
			}
		})
	}
}

func TestEnforce(t *testing.T) {
	policyDuration := 1 * time.Hour
	policy := newTokenPolicy(policyDuration, map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
	})

	for name, tc := range map[string]struct {
		req          *v1.GenerateTokenForPermissionsAndScopeRequest
		clusterID    string
		expectError  bool
		expectCapped bool
	}{
		"console permission set without scope": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName},
				HttpRequestScopes:  []*apiV1.HTTPRequestScope{{Path: "/api/graphql", Methods: []string{"POST"}}},
				Lifetime:           durationpb.New(30 * time.Minute),
			},
			clusterID: fixtureconsts.Cluster1,
		},
		"console permission set can request approved scope": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName},
				HttpRequestScopes:  []*apiV1.HTTPRequestScope{{Path: "/v1/metadata", Methods: []string{"GET"}}},
				Lifetime:           durationpb.New(30 * time.Minute),
			},
			clusterID: fixtureconsts.Cluster1,
		},
		"legacy permissions can request independent scope": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:       map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}},
				Lifetime:          durationpb.New(30 * time.Minute),
			},
			clusterID: fixtureconsts.Cluster1,
		},
		"legacy permissions reject invalid scope": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:       map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/%6detrics", Methods: []string{"GET"}}},
				Lifetime:          durationpb.New(30 * time.Minute),
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"console permission set with namespace scope": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName},
				HttpRequestScopes:  []*apiV1.HTTPRequestScope{{Path: "/api/graphql", Methods: []string{"POST"}}},
				ClusterScopes:      []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1, Namespaces: []string{"ns"}}},
				Lifetime:           durationpb.New(30 * time.Minute),
			},
			clusterID: fixtureconsts.Cluster1,
		},
		"console permission set with full cluster scope": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName},
				HttpRequestScopes:  []*apiV1.HTTPRequestScope{{Path: "/api/graphql", Methods: []string{"POST"}}},
				ClusterScopes:      []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1, FullClusterAccess: true}},
				Lifetime:           durationpb.New(30 * time.Minute),
			},
			clusterID: fixtureconsts.Cluster1,
		},
		"console permission set cannot access other cluster": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName},
				ClusterScopes:      []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster2, FullClusterAccess: true}},
				Lifetime:           durationpb.New(30 * time.Minute),
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"console permission set ignores fallback permissions": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName},
				HttpRequestScopes:  []*apiV1.HTTPRequestScope{{Path: "/api/graphql", Methods: []string{"POST"}}},
				Permissions:        map[string]v1.Access{"NetworkGraph": v1.Access_READ_WRITE_ACCESS},
				Lifetime:           durationpb.New(30 * time.Minute),
			},
			clusterID: fixtureconsts.Cluster1,
		},
		"generic permission set with cluster scope": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{"unknown"},
				ClusterScopes:      []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1, FullClusterAccess: true}},
				HttpRequestScopes:  []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}},
				Lifetime:           durationpb.New(30 * time.Minute),
			},
			clusterID: fixtureconsts.Cluster1,
		},
		"generic permission set without cluster scope": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSetNames: []string{"unknown"},
				HttpRequestScopes:  []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}},
				Lifetime:           durationpb.New(30 * time.Minute),
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"permission validation failure": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"NetworkGraph": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
				Lifetime:      durationpb.New(5 * time.Minute),
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"cluster scope violation": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster2}},
				Lifetime:      durationpb.New(5 * time.Minute),
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"nil lifetime": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"invalid proto duration": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
				Lifetime:      &durationpb.Duration{Seconds: 60, Nanos: -654321987},
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"zero lifetime": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
				Lifetime:      durationpb.New(0),
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"negative lifetime": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
				Lifetime:      durationpb.New(-5 * time.Minute),
			},
			clusterID:   fixtureconsts.Cluster1,
			expectError: true,
		},
		"lifetime within limit": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
				Lifetime:      durationpb.New(30 * time.Minute),
			},
			clusterID: fixtureconsts.Cluster1,
		},
		"lifetime exceeds limit": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
				Lifetime:      durationpb.New(2 * time.Hour),
			},
			clusterID:    fixtureconsts.Cluster1,
			expectCapped: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := sensorContext(t, gomock.NewController(t), tc.clusterID)
			result, _, err := policy.enforce(ctx, tc.req)
			if tc.expectError {
				assert.ErrorIs(t, err, errox.InvalidArgs)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tc.expectCapped {
					assert.Equal(t, durationpb.New(policyDuration), result.GetLifetime())
				} else {
					assert.Equal(t, tc.req.GetLifetime(), result.GetLifetime())
				}
			}
		})
	}

	t.Run("missing identity rejects request", func(t *testing.T) {
		req := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
			ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
			Lifetime:      durationpb.New(5 * time.Minute),
		}
		result, _, err := policy.enforce(t.Context(), req)
		assert.ErrorIs(t, err, errox.NotAuthorized)
		assert.Nil(t, result)
	})

	t.Run("identity with nil service rejects request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockIdentity := authnMocks.NewMockIdentity(ctrl)
		mockIdentity.EXPECT().Service().Return(nil).AnyTimes()
		ctx := authn.ContextWithIdentity(t.Context(), mockIdentity, t)

		req := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
			ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
			Lifetime:      durationpb.New(5 * time.Minute),
		}
		result, _, err := policy.enforce(ctx, req)
		assert.ErrorIs(t, err, errox.NotAuthorized)
		assert.Nil(t, result)
	})

	t.Run("non-sensor service identity rejects request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockIdentity := authnMocks.NewMockIdentity(ctrl)
		mockIdentity.EXPECT().Service().Return(&storage.ServiceIdentity{
			Id:   "some-service-id",
			Type: storage.ServiceType_CENTRAL_SERVICE,
		}).AnyTimes()
		ctx := authn.ContextWithIdentity(t.Context(), mockIdentity, t)

		req := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
			ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
			Lifetime:      durationpb.New(5 * time.Minute),
		}
		result, _, err := policy.enforce(ctx, req)
		assert.ErrorIs(t, err, errox.NotAuthorized)
		assert.Nil(t, result)
	})

	t.Run("zero maxLifetime applies no cap", func(t *testing.T) {
		p := newTokenPolicy(0, map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS})
		req := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
			ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
			Lifetime:      durationpb.New(2 * time.Hour),
		}
		ctx := sensorContext(t, gomock.NewController(t), fixtureconsts.Cluster1)
		result, _, err := p.enforce(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2*time.Hour, result.GetLifetime().AsDuration())
	})

	t.Run("negative maxLifetime applies no cap", func(t *testing.T) {
		p := newTokenPolicy(-policyDuration, map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS})
		req := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
			ClusterScopes: []*v1.ClusterScope{{ClusterId: fixtureconsts.Cluster1}},
			Lifetime:      durationpb.New(2 * time.Hour),
		}
		ctx := sensorContext(t, gomock.NewController(t), fixtureconsts.Cluster1)
		result, _, err := p.enforce(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2*time.Hour, result.GetLifetime().AsDuration())
	})
}
