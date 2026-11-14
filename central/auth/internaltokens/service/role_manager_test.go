package service

import (
	"context"
	"errors"
	"testing"

	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func expectPermissionSetLookup(t *testing.T, roles *roleMocks.MockDataStore, names []string, matches []*storage.PermissionSet, lookupErr error) {
	t.Helper()
	roles.EXPECT().GetPermissionSetsFiltered(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filter func(*storage.PermissionSet) bool) ([]*storage.PermissionSet, error) {
			for _, name := range names {
				assert.True(t, filter(&storage.PermissionSet{Name: name}))
			}
			assert.False(t, filter(&storage.PermissionSet{Name: "unrequested permission set"}))
			return matches, lookupErr
		})
}

func TestCreateSensorRole(t *testing.T) {
	for name, tc := range map[string]struct {
		scopes   []*v1.ClusterScope
		clusters tokens.ClusterScopes
	}{
		"authentication only": {},
		"namespace": {
			scopes:   []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"prod"}}},
			clusters: tokens.ClusterScopes{"sensor-cluster": {"prod"}},
		},
		"full cluster": {
			scopes:   []*v1.ClusterScope{{ClusterId: "sensor-cluster", FullClusterAccess: true}},
			clusters: tokens.ClusterScopes{"sensor-cluster": {"*"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			roles := roleMocks.NewMockDataStore(ctrl)
			expectPermissionSetLookup(t, roles, []string{centralproxy.ConsoleProxyPermissionSetName},
				[]*storage.PermissionSet{{
					Name: centralproxy.ConsoleProxyPermissionSetName,
					ResourceToAccess: map[string]storage.Access{
						"Deployment": storage.Access_READ_ACCESS,
						"Image":      storage.Access_READ_ACCESS,
					},
					Traits: &storage.Traits{Origin: storage.Traits_DEFAULT},
				}}, nil)

			role, err := (&roleManager{roles}).createSensorRole(t.Context(),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					ClusterScopes:      tc.scopes,
					PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName},
				},
			)
			require.NoError(t, err)
			assert.Equal(t, &tokens.InternalRole{
				RoleName: internalRoleName,
				Clusters: tc.clusters,
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {"Deployment", "Image"},
				},
			}, role)
		})
	}
}

func TestCreateSensorRoleFailsClosed(t *testing.T) {
	manager := &roleManager{}
	_, err := manager.createSensorRole(t.Context(), nil)
	require.ErrorIs(t, err, errox.ServerError)
	_, err = manager.createSensorRole(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
		PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName}})
	require.ErrorIs(t, err, errox.ServerError)

	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	expectPermissionSetLookup(t, roles, []string{centralproxy.ConsoleProxyPermissionSetName}, nil, nil)
	manager.roleStore = roles
	_, err = manager.createSensorRole(t.Context(), &v1.GenerateTokenForPermissionsAndScopeRequest{
		PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName}})
	require.ErrorIs(t, err, errox.NotFound)
}

func TestCreateSensorRoleResolvesAdministratorSetByName(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	expectPermissionSetLookup(t, roles, []string{centralproxy.ConsoleProxyPermissionSetName}, []*storage.PermissionSet{{
		Id:   "5e151917-5903-4701-9658-8fe4044c722f",
		Name: centralproxy.ConsoleProxyPermissionSetName,
		ResourceToAccess: map[string]storage.Access{
			"Deployment": storage.Access_READ_ACCESS,
		},
	}}, nil)
	role, err := (&roleManager{roles}).createSensorRole(t.Context(),
		&v1.GenerateTokenForPermissionsAndScopeRequest{
			PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName}})
	require.NoError(t, err)
	assert.Equal(t, []string{"Deployment"}, role.Permissions[storage.Access_READ_ACCESS])
}

func TestCreateSensorRoleUsesStoredPermissions(t *testing.T) {
	for name, tc := range map[string]struct {
		permissions map[string]storage.Access
		expected    map[storage.Access][]string
	}{
		"global permission": {
			permissions: map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_ACCESS,
				"Access":     storage.Access_READ_ACCESS,
			},
			expected: map[storage.Access][]string{storage.Access_READ_ACCESS: {"Access", "Deployment", "Image"}},
		},
		"write permission": {
			permissions: map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_WRITE_ACCESS,
			},
			expected: map[storage.Access][]string{
				storage.Access_READ_ACCESS:       {"Deployment"},
				storage.Access_READ_WRITE_ACCESS: {"Image"},
			},
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
			roles := roleMocks.NewMockDataStore(gomock.NewController(t))
			expectPermissionSetLookup(t, roles, []string{centralproxy.ConsoleProxyPermissionSetName},
				[]*storage.PermissionSet{{Name: centralproxy.ConsoleProxyPermissionSetName, ResourceToAccess: tc.permissions}}, nil)
			role, err := (&roleManager{roles}).createSensorRole(t.Context(),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName}})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, role.Permissions)
		})
	}
}

func TestCreateSensorRoleCombinesPermissionSets(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	expectPermissionSetLookup(t, roles,
		[]string{centralproxy.ConsoleProxyPermissionSetName, centralproxy.MetricsPermissionSetName},
		[]*storage.PermissionSet{
			{
				Name: centralproxy.ConsoleProxyPermissionSetName,
				ResourceToAccess: map[string]storage.Access{
					"Deployment":   storage.Access_READ_ACCESS,
					"Image":        storage.Access_READ_WRITE_ACCESS,
					"Access":       storage.Access_NO_ACCESS,
					"NetworkGraph": storage.Access_READ_ACCESS,
				},
			},
			{
				Name: centralproxy.MetricsPermissionSetName,
				ResourceToAccess: map[string]storage.Access{
					"Deployment":     storage.Access_READ_WRITE_ACCESS,
					"Image":          storage.Access_READ_ACCESS,
					"Access":         storage.Access_READ_ACCESS,
					"Administration": storage.Access_READ_ACCESS,
				},
			},
		}, nil)

	role, err := (&roleManager{roles}).createSensorRole(t.Context(),
		&v1.GenerateTokenForPermissionsAndScopeRequest{
			ClusterScopes:      []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"*"}}},
			PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName, centralproxy.MetricsPermissionSetName},
		})
	require.NoError(t, err)
	assert.Equal(t, &tokens.InternalRole{
		RoleName: internalRoleName,
		Clusters: tokens.ClusterScopes{"sensor-cluster": {"*"}},
		Permissions: map[storage.Access][]string{
			storage.Access_READ_ACCESS:       {"Access", "Administration", "NetworkGraph"},
			storage.Access_READ_WRITE_ACCESS: {"Deployment", "Image"},
		},
	}, role)
}

func TestCreateSensorRoleFailsIfAnyPermissionSetCannotBeLoaded(t *testing.T) {
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
				Name: centralproxy.ConsoleProxyPermissionSetName,
				ResourceToAccess: map[string]storage.Access{
					"Deployment": storage.Access_READ_ACCESS,
				},
			}}
			matches = append(matches, tc.permissionSets...)
			expectPermissionSetLookup(t, roles,
				[]string{centralproxy.ConsoleProxyPermissionSetName, centralproxy.MetricsPermissionSetName}, matches, tc.storeErr)
			role, err := (&roleManager{roles}).createSensorRole(t.Context(),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSetNames: []string{centralproxy.ConsoleProxyPermissionSetName, centralproxy.MetricsPermissionSetName}})
			assert.Nil(t, role)
			if tc.storeErr != nil {
				require.ErrorIs(t, err, tc.storeErr)
			} else {
				require.ErrorIs(t, err, tc.expectedErr)
			}
		})
	}
}
