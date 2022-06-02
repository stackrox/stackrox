package store

// ClusterStore stores the network edges per cluster.
type ClusterStore interface {
	GetFlowStore(clusterID string) FlowStore
}

//go:generate mockgen-wrapper
