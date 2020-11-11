package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkgraph/aggregator"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
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

	var networkTree tree.ReadOnlyNetworkTree
	if features.NetworkGraphExternalSrcs.Enabled() {
		networkTree = cds.networkTreeMgr.GetReadOnlyNetworkTree(clusterID)
		if networkTree == nil {
			networkTree = cds.networkTreeMgr.CreateNetworkTree(clusterID)
		}
	}

	return &flowDataStoreImpl{
		storage:                   cds.storage.GetFlowStore(clusterID),
		graphConfig:               cds.graphConfig,
		hideDefaultExtSrcsManager: aggregator.NewDefaultToCustomExtSrcConnAggregator(networkTree),
		deletedDeploymentsCache:   cds.deletedDeploymentsCache,
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

	var networkTree tree.ReadOnlyNetworkTree
	if features.NetworkGraphExternalSrcs.Enabled() {
		networkTree = cds.networkTreeMgr.GetReadOnlyNetworkTree(clusterID)
		if networkTree == nil {
			networkTree = cds.networkTreeMgr.CreateNetworkTree(clusterID)
		}
	}

	return &flowDataStoreImpl{
		storage:                   underlying,
		graphConfig:               cds.graphConfig,
		hideDefaultExtSrcsManager: aggregator.NewDefaultToCustomExtSrcConnAggregator(networkTree),
		deletedDeploymentsCache:   cds.deletedDeploymentsCache,
	}, nil
}
