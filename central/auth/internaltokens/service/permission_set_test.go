package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
)

const testPermissionSetID = "b3070020-ecc3-4f34-a3f6-ad28ffc9b80e"

func permissionSetRequest(byID bool) *v1.GenerateTokenForPermissionsAndScopeRequest {
	req := &v1.GenerateTokenForPermissionsAndScopeRequest{
		PermissionSet: &v1.GenerateTokenForPermissionsAndScopeRequest_PermissionSetName{PermissionSetName: "Reader"},
		Lifetime:      durationpb.New(time.Hour),
		ClusterScopes: []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"prod"}}},
		Requester:     "test requester",
	}
	if byID {
		req.PermissionSet = &v1.GenerateTokenForPermissionsAndScopeRequest_PermissionSetId{PermissionSetId: testPermissionSetID}
	}
	return req
}

func constrainedPermissionSet() *storage.PermissionSet {
	return &storage.PermissionSet{
		Id:               testPermissionSetID,
		Name:             "Reader",
		AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
		ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
		ResourceToAccess: map[string]storage.Access{"Node": storage.Access_READ_ACCESS, "Deployment": storage.Access_READ_ACCESS},
	}
}

func TestPermissionSetPolicy(t *testing.T) {
	for name, tc := range map[string]struct {
		change func(*v1.GenerateTokenForPermissionsAndScopeRequest)
		err    error
	}{
		"name": {},
		"id": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.PermissionSet = permissionSetRequest(true).GetPermissionSet()
		}},
		"empty name": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.PermissionSet = &v1.GenerateTokenForPermissionsAndScopeRequest_PermissionSetName{}
		}, err: errox.InvalidArgs},
		"empty id": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.PermissionSet = &v1.GenerateTokenForPermissionsAndScopeRequest_PermissionSetId{}
		}, err: errox.InvalidArgs},
		"invalid id": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.PermissionSet = &v1.GenerateTokenForPermissionsAndScopeRequest_PermissionSetId{PermissionSetId: "invalid"}
		}, err: errox.InvalidArgs},
		"surrounding whitespace": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.PermissionSet = &v1.GenerateTokenForPermissionsAndScopeRequest_PermissionSetName{PermissionSetName: " Reader "}
		}, err: errox.InvalidArgs},
		"mixed selector and legacy permissions": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.Permissions = map[string]v1.Access{"Deployment": v1.Access_READ_ACCESS}
		}, err: errox.InvalidArgs},
		"foreign cluster": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.ClusterScopes[0].ClusterId = "another-cluster"
		}, err: errox.InvalidArgs},
		"nil cluster scope": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.ClusterScopes = []*v1.ClusterScope{nil}
		}, err: errox.InvalidArgs},
		"empty scope remains valid": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.ClusterScopes = nil
		}},
		"missing lifetime": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.Lifetime = nil
		}, err: errox.InvalidArgs},
		"zero lifetime": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.Lifetime = durationpb.New(0)
		}, err: errox.InvalidArgs},
		"capped lifetime": {change: func(req *v1.GenerateTokenForPermissionsAndScopeRequest) {
			req.Lifetime = durationpb.New(2 * time.Hour)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			req := permissionSetRequest(false)
			if tc.change != nil {
				tc.change(req)
			}
			original := req.CloneVT()
			ctx := sensorContext(t, gomock.NewController(t), "sensor-cluster")
			result, err := defaultTokenPolicy().enforce(ctx, req)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, time.Hour, result.GetLifetime().AsDuration())
			assert.Equal(t, original.GetPermissionSet(), result.GetPermissionSet())
			protoassert.Equal(t, original, req, "lifetime capping must not mutate the request")
		})
	}
}

func TestPermissionSetPolicyRequiresSensorIdentity(t *testing.T) {
	for name, byID := range map[string]bool{"name": false, "id": true} {
		t.Run(name, func(t *testing.T) {
			req := permissionSetRequest(byID)
			_, err := defaultTokenPolicy().enforce(t.Context(), req)
			require.ErrorIs(t, err, errox.NotAuthorized)
			_, err = defaultTokenPolicy().enforce(sensorContext(t, gomock.NewController(t), ""), req)
			require.ErrorIs(t, err, errox.NotAuthorized)
		})
	}
}

func TestPermissionSetSelectorPresenceOnWire(t *testing.T) {
	for name, req := range map[string]*v1.GenerateTokenForPermissionsAndScopeRequest{
		"absent selector": {},
		"empty name":      {PermissionSet: &v1.GenerateTokenForPermissionsAndScopeRequest_PermissionSetName{}},
		"empty id":        {PermissionSet: &v1.GenerateTokenForPermissionsAndScopeRequest_PermissionSetId{}},
	} {
		t.Run(name, func(t *testing.T) {
			req.Lifetime = durationpb.New(time.Hour)
			encoded, err := req.MarshalVT()
			require.NoError(t, err)
			decoded := new(v1.GenerateTokenForPermissionsAndScopeRequest)
			require.NoError(t, decoded.UnmarshalVT(encoded))
			assert.Equal(t, req.GetPermissionSet(), decoded.GetPermissionSet())
			ctx := sensorContext(t, gomock.NewController(t), "sensor-cluster")
			_, err = defaultTokenPolicy().enforce(ctx, decoded)
			if req.PermissionSet == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, errox.InvalidArgs)
			}
		})
	}
}

func TestPermissionSetRoleRejectsUnsafeSets(t *testing.T) {
	for name, change := range map[string]func(*storage.PermissionSet){
		"missing read constraint": func(ps *storage.PermissionSet) {
			ps.AccessConstraint = storage.AccessConstraint_ACCESS_CONSTRAINT_UNSPECIFIED
		},
		"missing scope constraint": func(ps *storage.PermissionSet) {
			ps.ScopeConstraint = storage.ScopeConstraint_SCOPE_CONSTRAINT_UNSPECIFIED
		},
		"missing both": func(ps *storage.PermissionSet) {
			ps.AccessConstraint = storage.AccessConstraint_ACCESS_CONSTRAINT_UNSPECIFIED
			ps.ScopeConstraint = storage.ScopeConstraint_SCOPE_CONSTRAINT_UNSPECIFIED
		},
		"unknown read constraint": func(ps *storage.PermissionSet) {
			ps.AccessConstraint = storage.AccessConstraint_ACCESS_CONSTRAINT_READ + 1
		},
		"unknown scope constraint": func(ps *storage.PermissionSet) {
			ps.ScopeConstraint = storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE + 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			store := roleMocks.NewMockDataStore(gomock.NewController(t))
			ps := constrainedPermissionSet()
			change(ps)
			store.EXPECT().GetPermissionSet(gomock.Any(), testPermissionSetID).Return(ps, true, nil)
			role, err := (&roleManager{roleStore: store}).createRoleForRoxClaims(t.Context(), permissionSetRequest(true))
			require.ErrorIs(t, err, errox.InvalidArgs)
			assert.Nil(t, role)
		})
	}
}

func TestPermissionSetLookupFailsClosed(t *testing.T) {
	for name, tc := range map[string]struct {
		byID      bool
		matches   []*storage.PermissionSet
		lookupErr error
		wantErr   error
	}{
		"missing id":          {byID: true, wantErr: errox.NotFound},
		"nil id result":       {byID: true, matches: []*storage.PermissionSet{nil}, wantErr: errox.NotFound},
		"missing name":        {wantErr: errox.NotFound},
		"nil name result":     {matches: []*storage.PermissionSet{nil}, wantErr: errox.NotFound},
		"ambiguous name":      {matches: []*storage.PermissionSet{constrainedPermissionSet(), constrainedPermissionSet()}, wantErr: errox.NotFound},
		"id lookup failure":   {byID: true, lookupErr: errors.New("id lookup failure")},
		"name lookup failure": {lookupErr: errors.New("name lookup failure")},
	} {
		t.Run(name, func(t *testing.T) {
			store := roleMocks.NewMockDataStore(gomock.NewController(t))
			if tc.byID {
				var ps *storage.PermissionSet
				if len(tc.matches) > 0 {
					ps = tc.matches[0]
				}
				store.EXPECT().GetPermissionSet(gomock.Any(), testPermissionSetID).Return(ps, len(tc.matches) > 0, tc.lookupErr)
			} else {
				store.EXPECT().GetPermissionSetsFiltered(gomock.Any(), gomock.Any()).Return(tc.matches, tc.lookupErr)
			}
			role, err := (&roleManager{roleStore: store}).createRoleForRoxClaims(t.Context(), permissionSetRequest(tc.byID))
			if tc.lookupErr != nil {
				require.ErrorIs(t, err, tc.lookupErr)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
			assert.Nil(t, role)
		})
	}
	_, err := (&roleManager{}).createRoleForRoxClaims(t.Context(), permissionSetRequest(true))
	require.ErrorIs(t, err, errox.ServerError)
}

func TestPermissionSetTokenRoundTrip(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	factory, validator, err := tokens.CreateIssuerFactoryAndValidator(
		"https://stackrox.io/jwt", key, "test-key", tokens.WithDefaultTTL(time.Hour))
	require.NoError(t, err)
	issuer := getTokenIssuer(factory)
	now := time.Now().Truncate(time.Second)
	for name, tc := range map[string]struct {
		byID     bool
		scopes   []*v1.ClusterScope
		clusters tokens.ClusterScopes
		emptySet bool
	}{
		"name and namespace": {
			scopes:   []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"prod"}}},
			clusters: tokens.ClusterScopes{"sensor-cluster": {"prod"}},
		},
		"id and full cluster": {
			byID:     true,
			scopes:   []*v1.ClusterScope{{ClusterId: "sensor-cluster", FullClusterAccess: true}},
			clusters: tokens.ClusterScopes{"sensor-cluster": {"*"}},
		},
		"console bootstrap empty scope": {},
		"empty permission set":          {byID: true, emptySet: true},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := roleMocks.NewMockDataStore(ctrl)
			ps := constrainedPermissionSet()
			if tc.emptySet {
				ps.ResourceToAccess = nil
			}
			checkLookupContext := func(ctx context.Context) {
				allowed, err := sac.ForResource(resources.Access).ReadAllowed(ctx)
				require.NoError(t, err)
				assert.True(t, allowed)
				allowed, err = sac.ForResource(resources.Access).WriteAllowed(ctx)
				require.NoError(t, err)
				assert.False(t, allowed)
				allowed, err = sac.ForResource(resources.Administration).ReadAllowed(ctx)
				require.NoError(t, err)
				assert.False(t, allowed)
			}
			if tc.byID {
				store.EXPECT().GetPermissionSet(gomock.Any(), ps.GetId()).DoAndReturn(
					func(ctx context.Context, _ string) (*storage.PermissionSet, bool, error) {
						checkLookupContext(ctx)
						return ps, true, nil
					})
			} else {
				store.EXPECT().GetPermissionSetsFiltered(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, filter func(*storage.PermissionSet) bool) ([]*storage.PermissionSet, error) {
						checkLookupContext(ctx)
						assert.True(t, filter(ps))
						assert.False(t, filter(nil))
						assert.False(t, filter(&storage.PermissionSet{Name: "Another Reader"}))
						return []*storage.PermissionSet{ps}, nil
					})
			}
			svc := &serviceImpl{issuer: issuer, roleManager: &roleManager{roleStore: store}, policy: defaultTokenPolicy(), now: func() time.Time { return now }}
			req := permissionSetRequest(tc.byID)
			req.ClusterScopes = tc.scopes
			req.Lifetime = durationpb.New(2 * time.Hour)
			response, err := svc.GenerateTokenForPermissionsAndScope(sensorContext(t, ctrl, "sensor-cluster"), req)
			require.NoError(t, err)
			token, err := validator.Validate(t.Context(), response.GetToken())
			require.NoError(t, err)
			assert.Equal(t, []string{internalTokenId}, []string(token.Audience))
			assert.Equal(t, req.GetRequester(), token.Requester)
			assert.Equal(t, now.Add(time.Hour), token.Expiry())
			require.Len(t, token.InternalRoles, 1)
			assert.Equal(t, internalRoleName, token.InternalRoles[0].RoleName)
			assert.Equal(t, tc.clusters, token.InternalRoles[0].Clusters)
			if tc.emptySet {
				assert.Empty(t, token.InternalRoles[0].GetPermissions())
			} else {
				assert.Equal(t, ps.GetResourceToAccess(), token.InternalRoles[0].GetPermissions())
				assert.Equal(t, []string{"Deployment", "Node"}, token.InternalRoles[0].Permissions[storage.Access_READ_ACCESS])
			}
			assert.Empty(t, token.RoleNames)
		})
	}
}
