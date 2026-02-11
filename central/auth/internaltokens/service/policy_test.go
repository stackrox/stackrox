package service

import (
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
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

func TestEnforceClusterScope(t *testing.T) {
	policy := newTokenPolicy(0, nil)

	for name, tc := range map[string]struct {
		scopes          []*v1.ClusterScope
		sensorClusterID string
		expectError     bool
	}{
		"nil scopes": {
			scopes:          nil,
			sensorClusterID: "cluster-A",
		},
		"empty scopes": {
			scopes:          []*v1.ClusterScope{},
			sensorClusterID: "cluster-A",
		},
		"matching cluster": {
			scopes: []*v1.ClusterScope{
				{ClusterId: "cluster-A"},
			},
			sensorClusterID: "cluster-A",
		},
		"multiple matching clusters": {
			scopes: []*v1.ClusterScope{
				{ClusterId: "cluster-A"},
				{ClusterId: "cluster-A", Namespaces: []string{"ns1"}},
			},
			sensorClusterID: "cluster-A",
		},
		"mismatched cluster": {
			scopes: []*v1.ClusterScope{
				{ClusterId: "cluster-B"},
			},
			sensorClusterID: "cluster-A",
			expectError:     true,
		},
		"one matching, one mismatched": {
			scopes: []*v1.ClusterScope{
				{ClusterId: "cluster-A"},
				{ClusterId: "cluster-B"},
			},
			sensorClusterID: "cluster-A",
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

func TestEnforce(t *testing.T) {
	policy := newTokenPolicy(1*time.Hour, map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
	})

	for name, tc := range map[string]struct {
		req          *v1.GenerateTokenForPermissionsAndScopeRequest
		clusterID    string
		expectError  bool
		expectCapped bool
	}{
		"permission validation failure": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"NetworkGraph": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
				Lifetime:      durationpb.New(5 * time.Minute),
			},
			clusterID:   "cluster-A",
			expectError: true,
		},
		"cluster scope violation": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-B"}},
				Lifetime:      durationpb.New(5 * time.Minute),
			},
			clusterID:   "cluster-A",
			expectError: true,
		},
		"invalid proto duration": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
				Lifetime:      &durationpb.Duration{Seconds: 60, Nanos: -654321987},
			},
			clusterID:   "cluster-A",
			expectError: true,
		},
		"zero lifetime": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
				Lifetime:      durationpb.New(0),
			},
			clusterID:   "cluster-A",
			expectError: true,
		},
		"negative lifetime": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
				Lifetime:      durationpb.New(-5 * time.Minute),
			},
			clusterID:   "cluster-A",
			expectError: true,
		},
		"lifetime within limit": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
				Lifetime:      durationpb.New(30 * time.Minute),
			},
			clusterID: "cluster-A",
		},
		"lifetime exceeds limit": {
			req: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
				ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
				Lifetime:      durationpb.New(2 * time.Hour),
			},
			clusterID:    "cluster-A",
			expectCapped: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := sensorContext(t, gomock.NewController(t), tc.clusterID)
			result, err := policy.enforce(ctx, tc.req)
			if tc.expectError {
				assert.ErrorIs(t, err, errox.InvalidArgs)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tc.expectCapped {
					assert.Equal(t, durationpb.New(1*time.Hour), result.GetLifetime())
				} else {
					assert.Equal(t, tc.req.GetLifetime(), result.GetLifetime())
				}
			}
		})
	}

	t.Run("missing identity rejects request", func(t *testing.T) {
		req := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
			ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
			Lifetime:      durationpb.New(5 * time.Minute),
		}
		result, err := policy.enforce(t.Context(), req)
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
			ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
			Lifetime:      durationpb.New(5 * time.Minute),
		}
		result, err := policy.enforce(ctx, req)
		assert.ErrorIs(t, err, errox.NotAuthorized)
		assert.Nil(t, result)
	})

	t.Run("zero maxLifetime applies no cap", func(t *testing.T) {
		p := newTokenPolicy(0, map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS})
		req := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
			ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
			Lifetime:      durationpb.New(2 * time.Hour),
		}
		ctx := sensorContext(t, gomock.NewController(t), "cluster-A")
		result, err := p.enforce(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2*time.Hour, result.GetLifetime().AsDuration())
	})

	t.Run("negative maxLifetime applies no cap", func(t *testing.T) {
		p := newTokenPolicy(-1*time.Hour, map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS})
		req := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS},
			ClusterScopes: []*v1.ClusterScope{{ClusterId: "cluster-A"}},
			Lifetime:      durationpb.New(2 * time.Hour),
		}
		ctx := sensorContext(t, gomock.NewController(t), "cluster-A")
		result, err := p.enforce(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 2*time.Hour, result.GetLifetime().AsDuration())
	})
}
