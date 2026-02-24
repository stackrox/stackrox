package service

import (
	clusterDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"go.uber.org/mock/gomock"
)

func setClusterStoreExpectations(
	input *v1.GenerateTokenForPermissionsAndScopeRequest,
	mockClusterStore *clusterDataStoreMocks.MockDataStore,
) {
	for _, clusterScope := range input.GetClusterScopes() {
		clusterIdName := clusterScope.GetClusterId()
		mockClusterStore.EXPECT().
			GetClusterName(gomock.Any(), clusterIdName).
			Times(1).
			Return(clusterIdName, true, nil)
	}
}
