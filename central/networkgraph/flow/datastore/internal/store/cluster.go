package store

import "context"

// ClusterStore stores the network edges per cluster.
//
//go:generate mockgen-wrapper
type ClusterStore interface {
	GetFlowStore(clusterID string) FlowStore

	CreateFlowStore(ctx context.Context, clusterID string) (FlowStore, error)
	RemoveFlowStore(ctx context.Context, clusterID string) error
}

//go:generate mockgen-wrapper
