package datastore

import (
	"context"

	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/pkg/expiringcache"
)

// ClusterDataStore stores the network edges per cluster.
//go:generate mockgen-wrapper
type ClusterDataStore interface {
	GetFlowStore(ctx context.Context, clusterID string) FlowDataStore

	CreateFlowStore(ctx context.Context, clusterID string) (FlowDataStore, error)
	RemoveFlowStore(ctx context.Context, clusterID string) error
}

// NewClusterDataStore returns a new instance of ClusterDataStore using the input storage underneath.
func NewClusterDataStore(storage store.ClusterStore, deletedDeploymentsCache expiringcache.Cache) ClusterDataStore {
	return &clusterDataStoreImpl{
		storage:                 storage,
		deletedDeploymentsCache: deletedDeploymentsCache,
	}
}
