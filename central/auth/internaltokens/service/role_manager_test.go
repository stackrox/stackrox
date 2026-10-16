package service

import (
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

func TestCreateSensorConsoleRole(t *testing.T) {
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
				Return(&storage.PermissionSet{ResourceToAccess: map[string]storage.Access{
					"Deployment": storage.Access_READ_ACCESS,
					"Image":      storage.Access_READ_WRITE_ACCESS,
				}}, true, nil)

			role, err := (&roleManager{}).createSensorConsoleRole(t.Context(), roles, tc.scopes)
			require.NoError(t, err)
			assert.Equal(t, &tokens.InternalRole{
				RoleName: centralproxy.ConsoleProxyRole,
				Clusters: tc.clusters,
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS:       {"Deployment"},
					storage.Access_READ_WRITE_ACCESS: {"Image"},
				},
			}, role)
		})
	}
}

func TestCreateSensorConsoleRoleFailsClosed(t *testing.T) {
	manager := &roleManager{}
	_, err := manager.createSensorConsoleRole(t.Context(), nil, nil)
	require.ErrorIs(t, err, errox.NotAuthorized)

	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.ConsoleProxyPermissionSetID).Return(nil, false, nil)
	_, err = manager.createSensorConsoleRole(t.Context(), roles, nil)
	require.ErrorIs(t, err, errox.NotFound)
}
func TestCreateSensorMetricsRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.MetricsPermissionSetID).
		Return(&storage.PermissionSet{ResourceToAccess: map[string]storage.Access{
			"Deployment":     storage.Access_READ_ACCESS,
			"Administration": storage.Access_READ_WRITE_ACCESS,
		}}, true, nil)

	role, err := (&roleManager{}).createSensorMetricsRole(t.Context(), roles, "sensor-cluster")
	require.NoError(t, err)
	assert.Equal(t, &tokens.InternalRole{
		RoleName: centralproxy.SecuredClusterMetricsRole,
		Clusters: tokens.ClusterScopes{"sensor-cluster": {"*"}},
		Permissions: map[storage.Access][]string{
			storage.Access_READ_ACCESS:       {"Deployment"},
			storage.Access_READ_WRITE_ACCESS: {"Administration"},
		},
	}, role)
}

func TestCreateSensorMetricsRoleFailsClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.MetricsPermissionSetID).Return(nil, false, nil)
	manager := &roleManager{}
	_, err := manager.createSensorMetricsRole(t.Context(), roles, "sensor-cluster")
	require.ErrorIs(t, err, errox.NotFound)
	_, err = manager.createSensorMetricsRole(t.Context(), roles, "")
	require.ErrorIs(t, err, errox.NotAuthorized)
}
