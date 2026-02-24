package service

import (
	clusterDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"go.uber.org/mock/gomock"
)

func setClusterStoreExpectations(
	mockClusterStore *clusterDataStoreMocks.MockDataStore,
	clusterIdToNameMap map[string]string,
) {
	for clusterID, clusterName := range clusterIdToNameMap {
		mockClusterStore.EXPECT().
			GetClusterName(gomock.Any(), clusterID).
			Times(1).
			Return(clusterName, true, nil)
	}
}
