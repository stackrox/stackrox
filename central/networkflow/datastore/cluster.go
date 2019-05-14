package datastore

import (
	"context"

	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
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
		storage: storage,
	}
}
