package service

import (
	"context"
	"errors"
	"testing"

	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func expectPermissionSetLookup(t *testing.T, roles *roleMocks.MockDataStore, names []string, matches []*storage.PermissionSet, lookupErr error) {
	t.Helper()
	for _, permissionSet := range matches {
		if permissionSet != nil {
			permissionSet.AccessConstraint = storage.AccessConstraint_ACCESS_CONSTRAINT_READ
			permissionSet.ScopeConstraint = storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE
		}
	}
	roles.EXPECT().GetPermissionSetsFiltered(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filter func(*storage.PermissionSet) bool) ([]*storage.PermissionSet, error) {
			for _, name := range names {
				assert.True(t, filter(&storage.PermissionSet{Name: name}))
			}
			assert.False(t, filter(&storage.PermissionSet{Name: "unrequested permission set"}))
			return matches, lookupErr
		})
}

func TestCreateScopedRolesWithoutHTTPScopes(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	expectPermissionSetLookup(t, roles, []string{accesscontrol.ConsoleProxy},
		[]*storage.PermissionSet{{
			Name: accesscontrol.ConsoleProxy,
			ResourceToAccess: map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_ACCESS,
			},
			Traits: &storage.Traits{Origin: storage.Traits_DEFAULT},
		}}, nil)

	resolved, err := (&roleManager{roles}).createScopedRoles(t.Context(),
		&v1.GenerateTokenForPermissionsAndScopeRequest{
			ClusterScopes:  []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"prod"}}},
			PermissionSets: []*v1.ScopedPermissionSet{{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.ConsoleProxy}}},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []*tokens.InternalRole{{
		RoleName: internalRoleName,
		Permissions: map[storage.Access][]string{
			storage.Access_READ_ACCESS: {"Deployment", "Image"},
		},
		Clusters: tokens.ClusterScopes{"sensor-cluster": {"prod"}},
	}}, resolved)
}

func TestCreateScopedRolesKeepsPermissionsSeparate(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	expectPermissionSetLookup(t, roles, []string{"Deployment Reader", "Image Reader"}, []*storage.PermissionSet{
		{Name: "Image Reader", ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_ACCESS}},
		{Name: "Deployment Reader", ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS}},
	}, nil)
	access, err := (&roleManager{roles}).createScopedRoles(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
		ClusterScopes: []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"prod"}}},
		PermissionSets: []*v1.ScopedPermissionSet{
			{Selector: &v1.ScopedPermissionSet_Name{Name: "Deployment Reader"}, HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}}},
			{Selector: &v1.ScopedPermissionSet_Name{Name: "Image Reader"}, HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET"}}}},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []*tokens.InternalRole{
		{RoleName: internalRoleName, Permissions: map[storage.Access][]string{storage.Access_READ_ACCESS: {"Deployment"}},
			Clusters:          tokens.ClusterScopes{"sensor-cluster": {"prod"}},
			HTTPRequestScopes: tokens.HTTPRequestScopes{"/v1/deployments": {"GET"}}},
		{RoleName: internalRoleName, Permissions: map[storage.Access][]string{storage.Access_READ_ACCESS: {"Image"}},
			Clusters:          tokens.ClusterScopes{"sensor-cluster": {"prod"}},
			HTTPRequestScopes: tokens.HTTPRequestScopes{"/v1/images": {"GET"}}},
	}, access)
}

func TestSensorProxyPermissionSetRequiresBothConstraints(t *testing.T) {
	for name, tc := range map[string]struct {
		access storage.AccessConstraint
		scope  storage.ScopeConstraint
		valid  bool
	}{
		"both": {
			access: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			scope:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			valid:  true,
		},
		"neither": {},
		"read only": {
			access: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
		},
		"scoped only": {
			scope: storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
		},
	} {
		t.Run(name, func(t *testing.T) {
			roles := roleMocks.NewMockDataStore(gomock.NewController(t))
			roles.EXPECT().GetPermissionSetsFiltered(gomock.Any(), gomock.Any()).Return([]*storage.PermissionSet{{
				Name:             "Reader",
				ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
				AccessConstraint: tc.access,
				ScopeConstraint:  tc.scope,
			}}, nil)
			manager := &roleManager{roles}
			_, err := manager.createScopedRoles(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
				PermissionSets: []*v1.ScopedPermissionSet{{
					Selector:          &v1.ScopedPermissionSet_Name{Name: "Reader"},
					HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}},
				}},
			})
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, errox.InvalidArgs)
				assert.ErrorContains(t, err, "must constrain access to read and resources to cluster/namespace")
			}
		})
	}
}

func TestCreateScopedRolesFailsClosed(t *testing.T) {
	manager := &roleManager{}
	_, err := manager.createScopedRoles(t.Context(), nil)
	require.ErrorIs(t, err, errox.ServerError)
	_, err = manager.createScopedRoles(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
		PermissionSets: []*v1.ScopedPermissionSet{{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.ConsoleProxy}}}})
	require.ErrorIs(t, err, errox.ServerError)

	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	expectPermissionSetLookup(t, roles, []string{accesscontrol.ConsoleProxy}, nil, nil)
	manager.roleStore = roles
	_, err = manager.createScopedRoles(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
		PermissionSets: []*v1.ScopedPermissionSet{{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.ConsoleProxy}}}})
	require.ErrorIs(t, err, errox.NotFound)
}

func TestCreateScopedRolesResolvesConsoleSetByName(t *testing.T) {
	for name, id := range map[string]string{
		"administrator set": "5e151917-5903-4701-9658-8fe4044c722f",
		"default set":       accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy],
	} {
		t.Run(name, func(t *testing.T) {
			roles := roleMocks.NewMockDataStore(gomock.NewController(t))
			expectPermissionSetLookup(t, roles, []string{accesscontrol.ConsoleProxy}, []*storage.PermissionSet{{
				Id: id, Name: accesscontrol.ConsoleProxy,
				ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
			}}, nil)
			resolved, err := (&roleManager{roles}).createScopedRoles(t.Context(),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSets: []*v1.ScopedPermissionSet{{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.ConsoleProxy}}}})
			require.NoError(t, err)
			require.Len(t, resolved, 1)
			assert.Equal(t, []string{"Deployment"}, resolved[0].Permissions[storage.Access_READ_ACCESS])
		})
	}
}

func TestCreateScopedRolesResolvesByID(t *testing.T) {
	for name, tc := range map[string]struct {
		stored *storage.PermissionSet
		want   error
	}{
		"console default": {stored: &storage.PermissionSet{
			Id:               accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy],
			Name:             accesscontrol.ConsoleProxy,
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
		}},
		"missing": {want: errox.NotFound},
		"different name": {stored: &storage.PermissionSet{
			Id:               accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy],
			Name:             "unexpected set",
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			roles := roleMocks.NewMockDataStore(gomock.NewController(t))
			roles.EXPECT().GetPermissionSet(gomock.Any(), accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy]).
				Return(tc.stored, tc.stored != nil, nil)
			resolved, err := (&roleManager{roles}).createScopedRoles(t.Context(),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSets: []*v1.ScopedPermissionSet{{Selector: &v1.ScopedPermissionSet_Id{Id: accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy]}}}})
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
				assert.Nil(t, resolved)
				return
			}
			require.NoError(t, err)
			require.Len(t, resolved, 1)
			assert.Equal(t, []string{"Deployment"}, resolved[0].Permissions[storage.Access_READ_ACCESS])
		})
	}
}

func TestCreateScopedRolesRejectsInvalidID(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	_, err := (&roleManager{roles}).createScopedRoles(t.Context(),
		&v1.GenerateTokenForPermissionsAndScopeRequest{
			PermissionSets: []*v1.ScopedPermissionSet{{Selector: &v1.ScopedPermissionSet_Id{Id: "not-a-uuid"}}}})
	require.ErrorIs(t, err, errox.InvalidArgs)
}

func TestCreateScopedRolesResolvesMixedSelectors(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	expectPermissionSetLookup(t, roles, []string{"Image Reader"}, []*storage.PermissionSet{{
		Id: "image-id", Name: "Image Reader", ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_ACCESS},
	}}, nil)
	roles.EXPECT().GetPermissionSet(gomock.Any(), accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy]).
		Return(&storage.PermissionSet{
			Id: accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy], Name: accesscontrol.ConsoleProxy,
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
		}, true, nil)
	scoped, err := (&roleManager{roles}).createScopedRoles(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
		PermissionSets: []*v1.ScopedPermissionSet{
			{Selector: &v1.ScopedPermissionSet_Id{Id: accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy]},
				HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}}},
			{Selector: &v1.ScopedPermissionSet_Name{Name: "Image Reader"},
				HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/images", Methods: []string{"GET"}}}},
		},
	})
	require.NoError(t, err)
	require.Len(t, scoped, 2)
	assert.Equal(t, []string{"Deployment"}, scoped[0].Permissions[storage.Access_READ_ACCESS])
	assert.Equal(t, []string{"Image"}, scoped[1].Permissions[storage.Access_READ_ACCESS])
}

func TestCreateScopedRolesRejectsAliasByNameAndID(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	set := &storage.PermissionSet{
		Id:               "b3070020-ecc3-4f34-a3f6-ad28ffc9b80c",
		Name:             "Reader",
		AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
		ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
	}
	roles.EXPECT().GetPermissionSetsFiltered(gomock.Any(), gomock.Any()).Return([]*storage.PermissionSet{set}, nil)
	roles.EXPECT().GetPermissionSet(gomock.Any(), set.GetId()).Return(set, true, nil)
	_, err := (&roleManager{roles}).createScopedRoles(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
		PermissionSets: []*v1.ScopedPermissionSet{
			{Selector: &v1.ScopedPermissionSet_Name{Name: set.GetName()}, HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/a", Methods: []string{"GET"}}}},
			{Selector: &v1.ScopedPermissionSet_Id{Id: set.GetId()}, HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/b", Methods: []string{"GET"}}}},
		},
	})
	require.ErrorIs(t, err, errox.InvalidArgs)
}

func TestCreateScopedRolesAllowsConstrainedSetByIDOnOtherRoute(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	const metricsID = "b3070020-ecc3-4f34-a3f6-ad28ffc9b80c"
	roles.EXPECT().GetPermissionSet(gomock.Any(), metricsID).
		Return(&storage.PermissionSet{
			Id: metricsID, Name: accesscontrol.OpenShiftPrometheus,
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
		}, true, nil)
	resolved, err := (&roleManager{roles}).createScopedRoles(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
		ClusterScopes: []*v1.ClusterScope{{ClusterId: "sensor-cluster", FullClusterAccess: true}},
		PermissionSets: []*v1.ScopedPermissionSet{{
			Selector: &v1.ScopedPermissionSet_Id{Id: metricsID}, HttpRequestScopes: []*apiV1.HTTPRequestScope{{Path: "/v1/deployments", Methods: []string{"GET"}}},
		}},
	})
	require.NoError(t, err)
	require.Len(t, resolved, 1)
	assert.Equal(t, tokens.HTTPRequestScopes{"/v1/deployments": {"GET"}}, resolved[0].HTTPRequestScopes)
	assert.Equal(t, tokens.ClusterScopes{"sensor-cluster": {"*"}}, resolved[0].Clusters)
}

func TestCreateScopedRolesUsesStoredPermissions(t *testing.T) {
	for name, tc := range map[string]struct {
		permissions map[string]storage.Access
		expected    map[storage.Access][]string
		wantErr     bool
	}{
		"global permission": {
			permissions: map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_ACCESS,
				"Access":     storage.Access_READ_ACCESS,
			},
			wantErr: true,
		},
		"write permission": {
			permissions: map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_WRITE_ACCESS,
			},
			wantErr: true,
		},
		"missing permission": {
			permissions: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
			expected:    map[storage.Access][]string{storage.Access_READ_ACCESS: {"Deployment"}},
		},
		"no access": {
			permissions: map[string]storage.Access{"Deployment": storage.Access_NO_ACCESS},
			expected:    map[storage.Access][]string{storage.Access_NO_ACCESS: {"Deployment"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			roles := roleMocks.NewMockDataStore(ctrl)
			expectPermissionSetLookup(t, roles, []string{accesscontrol.ConsoleProxy},
				[]*storage.PermissionSet{{Name: accesscontrol.ConsoleProxy, ResourceToAccess: tc.permissions}}, nil)
			resolved, err := (&roleManager{roles}).createScopedRoles(t.Context(),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSets: []*v1.ScopedPermissionSet{{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.ConsoleProxy}}}})
			if tc.wantErr {
				require.ErrorIs(t, err, errox.InvalidArgs)
				assert.Nil(t, resolved)
				return
			}
			require.NoError(t, err)
			require.Len(t, resolved, 1)
			assert.Equal(t, tc.expected, resolved[0].Permissions)
		})
	}
}

func TestCreateScopedRolesKeepsUnscopedPermissionsSeparate(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	expectPermissionSetLookup(t, roles,
		[]string{accesscontrol.ConsoleProxy, accesscontrol.OpenShiftPrometheus},
		[]*storage.PermissionSet{
			{
				Name: accesscontrol.ConsoleProxy,
				ResourceToAccess: map[string]storage.Access{
					"Deployment":   storage.Access_READ_ACCESS,
					"Image":        storage.Access_NO_ACCESS,
					"Access":       storage.Access_NO_ACCESS,
					"NetworkGraph": storage.Access_READ_ACCESS,
				},
			},
			{
				Name: accesscontrol.OpenShiftPrometheus,
				ResourceToAccess: map[string]storage.Access{
					"Deployment":     storage.Access_READ_ACCESS,
					"Image":          storage.Access_READ_ACCESS,
					"Node":           storage.Access_READ_ACCESS,
					"Administration": storage.Access_NO_ACCESS,
				},
			},
		}, nil)

	resolved, err := (&roleManager{roles}).createScopedRoles(t.Context(),
		&v1.GenerateTokenForPermissionsAndScopeRequest{
			ClusterScopes: []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"*"}}},
			PermissionSets: []*v1.ScopedPermissionSet{
				{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.ConsoleProxy}},
				{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.OpenShiftPrometheus}},
			},
		})
	require.NoError(t, err)
	assert.Equal(t, []*tokens.InternalRole{
		{
			RoleName: internalRoleName,
			Permissions: map[storage.Access][]string{
				storage.Access_READ_ACCESS: {"Deployment", "NetworkGraph"},
				storage.Access_NO_ACCESS:   {"Access", "Image"},
			},
			Clusters: tokens.ClusterScopes{"sensor-cluster": {"*"}},
		},
		{
			RoleName: internalRoleName,
			Permissions: map[storage.Access][]string{
				storage.Access_READ_ACCESS: {"Deployment", "Image", "Node"},
				storage.Access_NO_ACCESS:   {"Administration"},
			},
			Clusters: tokens.ClusterScopes{"sensor-cluster": {"*"}},
		},
	}, resolved)
}

func TestCreateScopedRolesFailsIfAnyPermissionSetCannotBeLoaded(t *testing.T) {
	for name, tc := range map[string]struct {
		permissionSets []*storage.PermissionSet
		storeErr       error
		expectedErr    error
	}{
		"missing": {expectedErr: errox.NotFound},
		"nil":     {permissionSets: []*storage.PermissionSet{nil}, expectedErr: errox.NotFound},
		"error":   {storeErr: errors.New("store failure")},
	} {
		t.Run(name, func(t *testing.T) {
			roles := roleMocks.NewMockDataStore(gomock.NewController(t))
			matches := []*storage.PermissionSet{{
				Name: accesscontrol.ConsoleProxy,
				ResourceToAccess: map[string]storage.Access{
					"Deployment": storage.Access_READ_ACCESS,
				},
			}}
			matches = append(matches, tc.permissionSets...)
			expectPermissionSetLookup(t, roles,
				[]string{accesscontrol.ConsoleProxy, accesscontrol.OpenShiftPrometheus}, matches, tc.storeErr)
			resolved, err := (&roleManager{roles}).createScopedRoles(t.Context(),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSets: []*v1.ScopedPermissionSet{
						{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.ConsoleProxy}},
						{Selector: &v1.ScopedPermissionSet_Name{Name: accesscontrol.OpenShiftPrometheus}},
					}})
			assert.Nil(t, resolved)
			if tc.storeErr != nil {
				require.ErrorIs(t, err, tc.storeErr)
			} else {
				require.ErrorIs(t, err, tc.expectedErr)
			}
		})
	}
}

func TestCreateScopedRolesFromGenericPermissionSet(t *testing.T) {
	const permissionSetName = "custom-read-only-set"
	for name, tc := range map[string]struct {
		permissions map[string]storage.Access
		expected    map[storage.Access][]string
	}{
		"read scoped resources and deny global resources": {
			permissions: map[string]storage.Access{
				"Image": storage.Access_READ_ACCESS, "Node": storage.Access_READ_ACCESS,
				"Administration": storage.Access_NO_ACCESS,
			},
			expected: map[storage.Access][]string{
				storage.Access_READ_ACCESS: {"Image", "Node"}, storage.Access_NO_ACCESS: {"Administration"},
			},
		},
		"empty set": {expected: map[storage.Access][]string{}},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			roles := roleMocks.NewMockDataStore(ctrl)
			expectPermissionSetLookup(t, roles, []string{permissionSetName},
				[]*storage.PermissionSet{{Name: permissionSetName, ResourceToAccess: tc.permissions}}, nil)
			resolved, err := (&roleManager{roles}).createScopedRoles(t.Context(),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSets: []*v1.ScopedPermissionSet{{Selector: &v1.ScopedPermissionSet_Name{Name: permissionSetName}}},
					ClusterScopes:  []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"prod"}}},
				})
			require.NoError(t, err)
			assert.Equal(t, []*tokens.InternalRole{{
				RoleName:    internalRoleName,
				Permissions: tc.expected,
				Clusters:    tokens.ClusterScopes{"sensor-cluster": {"prod"}},
			}}, resolved)
		})
	}
}
