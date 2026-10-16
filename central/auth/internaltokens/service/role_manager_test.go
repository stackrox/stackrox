package service

import (
	"testing"

	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	permissionMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
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
			resolved := permissionMocks.NewMockResolvedRole(ctrl)
			roles.EXPECT().GetAndResolveRole(gomock.Any(), centralproxy.ConsoleProxyRole).Return(resolved, nil)
			resolved.EXPECT().GetPermissions().Return(map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_NO_ACCESS,
			})

			role, err := (&roleManager{}).createSensorConsoleRole(t.Context(), roles, centralproxy.ConsoleProxyRole, tc.scopes)
			require.NoError(t, err)
			assert.Equal(t, &tokens.InternalRole{
				RoleName: centralproxy.ConsoleProxyRole,
				Clusters: tc.clusters,
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {"Deployment"},
				},
			}, role)
		})
	}
}

func TestCreateSensorConsoleRoleRejectsWriteAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	resolved := permissionMocks.NewMockResolvedRole(ctrl)
	roles.EXPECT().GetAndResolveRole(gomock.Any(), centralproxy.ConsoleProxyRole).Return(resolved, nil)
	resolved.EXPECT().GetPermissions().Return(map[string]storage.Access{
		"Deployment": storage.Access_READ_WRITE_ACCESS,
	})

	_, err := (&roleManager{}).createSensorConsoleRole(t.Context(), roles, centralproxy.ConsoleProxyRole, nil)
	require.ErrorIs(t, err, errox.InvalidArgs)
}

func TestCreateSensorConsoleRoleFailsClosed(t *testing.T) {
	manager := &roleManager{}
	_, err := manager.createSensorConsoleRole(t.Context(), nil, centralproxy.ConsoleProxyRole, nil)
	require.ErrorIs(t, err, errox.NotAuthorized)

	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetAndResolveRole(gomock.Any(), centralproxy.ConsoleProxyRole).Return(nil, nil)
	_, err = manager.createSensorConsoleRole(t.Context(), roles, centralproxy.ConsoleProxyRole, nil)
	require.ErrorIs(t, err, errox.NotFound)
}
