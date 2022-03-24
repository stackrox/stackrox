package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkgraph/aggregator"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/sac"
)

type clusterDataStoreImpl struct {
	storage                 store.ClusterStore
	networkTreeMgr          networktree.Manager
	graphConfig             graphConfigDS.DataStore
	deletedDeploymentsCache expiringcache.Cache
}

func (cds *clusterDataStoreImpl) GetFlowStore(ctx context.Context, clusterID string) (FlowDataStore, error) {
	if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil || !ok {
		return nil, err
	}

	aggr, err := cds.getAggregator(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	return &flowDataStoreImpl{
		storage:                   cds.storage.GetFlowStore(clusterID),
		graphConfig:               cds.graphConfig,
		hideDefaultExtSrcsManager: aggr,
		deletedDeploymentsCache:   cds.deletedDeploymentsCache,
	}, nil
}

func (cds *clusterDataStoreImpl) CreateFlowStore(ctx context.Context, clusterID string) (FlowDataStore, error) {
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	underlying, err := cds.storage.CreateFlowStore(ctx, clusterID)

	if err != nil {
		return nil, err
	}

	aggr, err := cds.getAggregator(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	return &flowDataStoreImpl{
		storage:                   underlying,
		graphConfig:               cds.graphConfig,
		hideDefaultExtSrcsManager: aggr,
		deletedDeploymentsCache:   cds.deletedDeploymentsCache,
	}, nil
}

func (cds *clusterDataStoreImpl) getAggregator(ctx context.Context, clusterID string) (aggregator.NetworkConnsAggregator, error) {
	networkTree := cds.networkTreeMgr.GetReadOnlyNetworkTree(ctx, clusterID)
	if networkTree == nil {
		networkTree = cds.networkTreeMgr.CreateNetworkTree(ctx, clusterID)
	}

	aggr, err := aggregator.NewDefaultToCustomExtSrcConnAggregator(networkTree)
	if err != nil {
		return nil, errors.Wrap(err, "creating a hide default networks manager")
	}
	return aggr, nil
}
