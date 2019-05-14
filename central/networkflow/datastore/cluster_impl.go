package datastore

import (
	"context"

	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
)

type clusterDataStoreImpl struct {
	storage store.ClusterStore
}

func (cds *clusterDataStoreImpl) GetFlowStore(_ context.Context, clusterID string) FlowDataStore {
	return &flowDataStoreImpl{
		storage: cds.storage.GetFlowStore(clusterID),
	}
}

func (cds *clusterDataStoreImpl) CreateFlowStore(_ context.Context, clusterID string) (FlowDataStore, error) {
	underlying, err := cds.storage.CreateFlowStore(clusterID)
	if err != nil {
		return nil, err
	}
	return &flowDataStoreImpl{
		storage: underlying,
	}, nil
}

func (cds *clusterDataStoreImpl) RemoveFlowStore(_ context.Context, clusterID string) error {
	return cds.storage.RemoveFlowStore(clusterID)
}
