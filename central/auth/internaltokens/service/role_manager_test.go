package service

import (
	"context"
	"testing"

	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	permissionMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateSensorMetricsRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	resolved := permissionMocks.NewMockResolvedRole(ctrl)
	roles.EXPECT().GetAndResolveRole(gomock.Any(), "metrics reader").Return(resolved, nil).Times(2)
	resolved.EXPECT().GetPermissions().Return(map[string]storage.Access{
		"Deployment": storage.Access_READ_ACCESS,
		"Image":      storage.Access_NO_ACCESS,
	})
	resolved.EXPECT().GetPermissions().Return(map[string]storage.Access{
		"Administration": storage.Access_READ_WRITE_ACCESS,
	})

	manager := &roleManager{}
	role, err := manager.createSensorMetricsRole(context.Background(), roles, "metrics reader", "sensor-cluster")
	require.NoError(t, err)
	assert.Equal(t, &tokens.InternalRole{
		RoleName: "metrics reader",
		Clusters: tokens.ClusterScopes{"sensor-cluster": {"*"}},
		Permissions: map[storage.Access][]string{
			storage.Access_READ_ACCESS: {"Deployment"},
		},
	}, role)

	_, err = manager.createSensorMetricsRole(context.Background(), roles, "metrics reader", "sensor-cluster")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read permissions")
}

func TestCreateSensorMetricsRoleFailsClosed(t *testing.T) {
	ctrl := gomock.NewController(t)
	roles := roleMocks.NewMockDataStore(ctrl)
	roles.EXPECT().GetAndResolveRole(gomock.Any(), "missing role").Return(nil, nil)
	manager := &roleManager{}
	_, err := manager.createSensorMetricsRole(context.Background(), roles, "missing role", "sensor-cluster")
	require.Error(t, err)
	_, err = manager.createSensorMetricsRole(context.Background(), roles, "missing role", "")
	require.Error(t, err)
}
