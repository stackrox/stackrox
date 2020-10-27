package datastore

import (
	"context"

	"github.com/pkg/errors"
	graphConfigDS "github.com/stackrox/rox/central/networkflow/config/datastore"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/sac"
)

type clusterDataStoreImpl struct {
	storage                 store.ClusterStore
	graphConfig             graphConfigDS.DataStore
	deletedDeploymentsCache expiringcache.Cache
}

func (cds *clusterDataStoreImpl) GetFlowStore(ctx context.Context, clusterID string) (FlowDataStore, error) {
	if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil || !ok {
		return nil, err
	}

	return &flowDataStoreImpl{
		storage:                 cds.storage.GetFlowStore(clusterID),
		graphConfig:             cds.graphConfig,
		deletedDeploymentsCache: cds.deletedDeploymentsCache,
	}, nil
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
		storage:                 underlying,
		graphConfig:             cds.graphConfig,
		deletedDeploymentsCache: cds.deletedDeploymentsCache,
	}, nil
}
