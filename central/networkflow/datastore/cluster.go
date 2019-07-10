package datastore

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/pkg/expiringcache"
)

const (
	deletedDeploymentsCacheSize       = 10000
	deletedDeploymentsRetentionPeriod = 2 * time.Minute
)

// ClusterDataStore stores the network edges per cluster.
//go:generate mockgen-wrapper ClusterDataStore
type ClusterDataStore interface {
	GetFlowStore(ctx context.Context, clusterID string) FlowDataStore

	CreateFlowStore(ctx context.Context, clusterID string) (FlowDataStore, error)
	RemoveFlowStore(ctx context.Context, clusterID string) error
}

// NewClusterDataStore returns a new instance of ClusterDataStore using the input storage underneath.
func NewClusterDataStore(storage store.ClusterStore) ClusterDataStore {
	return &clusterDataStoreImpl{
		storage:                 storage,
		deletedDeploymentsCache: expiringcache.NewExpiringCache(deletedDeploymentsCacheSize, deletedDeploymentsRetentionPeriod),
	}
}
