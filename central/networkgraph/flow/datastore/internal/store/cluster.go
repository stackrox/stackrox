package store

import "context"

// ClusterStore stores the network edges per cluster.
type ClusterStore interface {
	GetFlowStore(clusterID string) FlowStore

	CreateFlowStore(ctx context.Context, clusterID string) (FlowStore, error)
}

//go:generate mockgen-wrapper
