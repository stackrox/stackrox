package service

import (
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
			roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.ConsoleProxyPermissionSetID).
				Return(&storage.PermissionSet{
					ResourceToAccess: map[string]storage.Access{
						"Deployment": storage.Access_READ_ACCESS,
						"Image":      storage.Access_READ_ACCESS,
					},
					Traits: &storage.Traits{Origin: storage.Traits_DEFAULT},
				}, true, nil)

			role, err := (&roleManager{roles}).createSensorRole(t.Context(),
				[]string{centralproxy.ConsoleProxyPermissionSetID},
				clusterScopesFromRequest(tc.scopes))
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
	_, err := manager.createSensorRole(t.Context(), nil, nil)
	require.ErrorIs(t, err, errox.InvalidArgs)
	_, err = manager.createSensorRole(t.Context(), []string{centralproxy.ConsoleProxyPermissionSetID}, nil)
	require.ErrorIs(t, err, errox.NotAuthorized)

	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.ConsoleProxyPermissionSetID).Return(nil, false, nil)
	manager.roleStore = roles
	_, err = manager.createSensorRole(t.Context(), []string{centralproxy.ConsoleProxyPermissionSetID}, nil)
	require.ErrorIs(t, err, errox.NotFound)
}

func TestCreateRoleFromConsolePermissionSetUsesStoredPermissions(t *testing.T) {
	for name, tc := range map[string]struct {
		permissions map[string]storage.Access
		expected    map[storage.Access][]string
		denied      bool
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
			denied: true,
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
			roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.ConsoleProxyPermissionSetID).
				Return(&storage.PermissionSet{ResourceToAccess: tc.permissions}, true, nil)
			role, err := (&roleManager{roles}).createRoleFromPermissionSets(sensorContext(t, ctrl, "sensor-cluster"),
				&v1.GenerateTokenForPermissionsAndScopeRequest{PermissionSetIds: []string{centralproxy.ConsoleProxyPermissionSetID}})
			if tc.denied {
				require.ErrorIs(t, err, errox.InvalidArgs)
				assert.Nil(t, role)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, role.Permissions)
		})
	}
}

func TestCreateSensorRoleCombinesPermissionSets(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.ConsoleProxyPermissionSetID).
		Return(&storage.PermissionSet{ResourceToAccess: map[string]storage.Access{
			"Deployment":   storage.Access_READ_ACCESS,
			"Image":        storage.Access_READ_WRITE_ACCESS,
			"Access":       storage.Access_NO_ACCESS,
			"NetworkGraph": storage.Access_READ_ACCESS,
		}}, true, nil)
	roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.MetricsPermissionSetID).
		Return(&storage.PermissionSet{ResourceToAccess: map[string]storage.Access{
			"Deployment":     storage.Access_READ_WRITE_ACCESS,
			"Image":          storage.Access_READ_ACCESS,
			"Access":         storage.Access_READ_ACCESS,
			"Administration": storage.Access_READ_ACCESS,
		}}, true, nil)

	role, err := (&roleManager{roles}).createSensorRole(t.Context(),
		[]string{centralproxy.ConsoleProxyPermissionSetID, centralproxy.MetricsPermissionSetID},
		tokens.ClusterScopes{"sensor-cluster": {"*"}})
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
		permissionSet *storage.PermissionSet
		found         bool
		storeErr      error
		expectedErr   error
	}{
		"missing": {expectedErr: errox.NotFound},
		"nil":     {found: true, expectedErr: errox.NotFound},
		"error":   {storeErr: errors.New("store failure")},
	} {
		t.Run(name, func(t *testing.T) {
			roles := roleMocks.NewMockDataStore(gomock.NewController(t))
			roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.ConsoleProxyPermissionSetID).
				Return(&storage.PermissionSet{ResourceToAccess: map[string]storage.Access{
					"Deployment": storage.Access_READ_ACCESS,
				}}, true, nil)
			roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.MetricsPermissionSetID).
				Return(tc.permissionSet, tc.found, tc.storeErr)
			role, err := (&roleManager{roles}).createSensorRole(t.Context(),
				[]string{centralproxy.ConsoleProxyPermissionSetID, centralproxy.MetricsPermissionSetID}, nil)
			assert.Nil(t, role)
			if tc.storeErr != nil {
				require.ErrorIs(t, err, tc.storeErr)
			} else {
				require.ErrorIs(t, err, tc.expectedErr)
			}
		})
	}
}

func TestCreateRoleFromMetricsPermissionSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.MetricsPermissionSetID).
		Return(&storage.PermissionSet{ResourceToAccess: map[string]storage.Access{
			"Deployment":     storage.Access_READ_ACCESS,
			"Administration": storage.Access_READ_ACCESS,
		}}, true, nil)

	role, err := (&roleManager{roles}).createRoleFromPermissionSets(sensorContext(t, ctrl, "sensor-cluster"),
		&v1.GenerateTokenForPermissionsAndScopeRequest{
			PermissionSetIds: []string{centralproxy.MetricsPermissionSetID},
			ClusterScopes:    []*v1.ClusterScope{{ClusterId: "sensor-cluster", FullClusterAccess: true}},
		})
	require.NoError(t, err)
	assert.Equal(t, &tokens.InternalRole{
		RoleName: internalRoleName,
		Clusters: tokens.ClusterScopes{"sensor-cluster": {"*"}},
		Permissions: map[storage.Access][]string{
			storage.Access_READ_ACCESS: {"Administration", "Deployment"},
		},
	}, role)
}

func TestCreateRoleFromMetricsPermissionSetRejectsStoredWriteOrUnknownAccess(t *testing.T) {
	for name, access := range map[string]storage.Access{
		"write":   storage.Access_READ_WRITE_ACCESS,
		"unknown": storage.Access(99),
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			roles := roleMocks.NewMockDataStore(ctrl)
			roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.MetricsPermissionSetID).
				Return(&storage.PermissionSet{ResourceToAccess: map[string]storage.Access{
					"Administration": access,
				}}, true, nil)
			role, err := (&roleManager{roles}).createRoleFromPermissionSets(sensorContext(t, ctrl, "sensor-cluster"),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSetIds: []string{centralproxy.MetricsPermissionSetID},
					ClusterScopes:    []*v1.ClusterScope{{ClusterId: "sensor-cluster", FullClusterAccess: true}},
				})
			assert.Nil(t, role)
			assert.ErrorIs(t, err, errox.InvalidArgs)
		})
	}
}

func TestCreateRoleFromMetricsPermissionSetFailsClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.MetricsPermissionSetID).Return(nil, false, nil)
	manager := &roleManager{roles}
	_, err := manager.createRoleFromPermissionSets(sensorContext(t, ctrl, "sensor-cluster"),
		&v1.GenerateTokenForPermissionsAndScopeRequest{
			PermissionSetIds: []string{centralproxy.MetricsPermissionSetID},
			ClusterScopes:    []*v1.ClusterScope{{ClusterId: "sensor-cluster", FullClusterAccess: true}},
		})
	require.ErrorIs(t, err, errox.NotFound)
}

func TestCreateRoleFromGenericPermissionSet(t *testing.T) {
	const permissionSetID = "custom-read-only-set"
	for name, tc := range map[string]struct {
		permissions map[string]storage.Access
		expected    map[storage.Access][]string
		invalid     bool
	}{
		"read and no access to arbitrary resources": {
			permissions: map[string]storage.Access{
				"Image": storage.Access_READ_ACCESS, "Administration": storage.Access_NO_ACCESS,
			},
			expected: map[storage.Access][]string{
				storage.Access_READ_ACCESS: {"Image"}, storage.Access_NO_ACCESS: {"Administration"},
			},
		},
		"empty set": {expected: map[storage.Access][]string{}},
		"write access": {
			permissions: map[string]storage.Access{"Image": storage.Access_READ_ACCESS, "Deployment": storage.Access_READ_WRITE_ACCESS},
			invalid:     true,
		},
		"unknown access": {
			permissions: map[string]storage.Access{"Image": storage.Access(99)},
			invalid:     true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			roles := roleMocks.NewMockDataStore(ctrl)
			roles.EXPECT().GetPermissionSet(gomock.Any(), permissionSetID).
				Return(&storage.PermissionSet{ResourceToAccess: tc.permissions}, true, nil)
			role, err := (&roleManager{roles}).createRoleFromPermissionSets(sensorContext(t, ctrl, "sensor-cluster"),
				&v1.GenerateTokenForPermissionsAndScopeRequest{
					PermissionSetIds: []string{permissionSetID},
					ClusterScopes:    []*v1.ClusterScope{{ClusterId: "sensor-cluster", Namespaces: []string{"prod"}}},
				})
			if tc.invalid {
				assert.Nil(t, role)
				assert.ErrorIs(t, err, errox.InvalidArgs)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, &tokens.InternalRole{
				RoleName:    internalRoleName,
				Clusters:    tokens.ClusterScopes{"sensor-cluster": {"prod"}},
				Permissions: tc.expected,
			}, role)
		})
	}
}

func TestCreateRoleFromGenericPermissionSetFailsClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := sensorContext(t, ctrl, "sensor-cluster")
	req := &v1.GenerateTokenForPermissionsAndScopeRequest{PermissionSetIds: []string{"missing"}}
	manager := &roleManager{}
	_, err := manager.createRoleFromPermissionSets(ctx, req)
	require.ErrorIs(t, err, errox.NotAuthorized)

	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetPermissionSet(gomock.Any(), "missing").Return(nil, false, nil)
	manager.roleStore = roles
	_, err = manager.createRoleFromPermissionSets(ctx, req)
	require.ErrorIs(t, err, errox.NotFound)
}
