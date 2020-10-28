package store

// ClusterStore stores the network edges per cluster.
type ClusterStore interface {
	GetFlowStore(clusterID string) FlowStore

	CreateFlowStore(clusterID string) (FlowStore, error)
}

//go:generate mockgen-wrapper
