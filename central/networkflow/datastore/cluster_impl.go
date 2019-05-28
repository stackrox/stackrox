package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
)

type clusterDataStoreImpl struct {
	storage store.ClusterStore
}

func (cds *clusterDataStoreImpl) GetFlowStore(ctx context.Context, clusterID string) FlowDataStore {
	if ok, err := networkGraphSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil
	}

	return &flowDataStoreImpl{
		storage: cds.storage.GetFlowStore(clusterID),
	}
}

func (cds *clusterDataStoreImpl) CreateFlowStore(ctx context.Context, clusterID string) (FlowDataStore, error) {
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, errors.New("permission denied")
	}

	underlying, err := cds.storage.CreateFlowStore(clusterID)
	if err != nil {
		return nil, err
	}
	return &flowDataStoreImpl{
		storage: underlying,
	}, nil
}

func (cds *clusterDataStoreImpl) RemoveFlowStore(ctx context.Context, clusterID string) error {
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return nil
	} else if !ok {
		return errors.New("permission denied")
	}

	return cds.storage.RemoveFlowStore(clusterID)
}
