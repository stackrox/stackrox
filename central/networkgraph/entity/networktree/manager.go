package networktree

import (
	"github.com/stackrox/rox/pkg/networkgraph/tree"
)

// Manager provides a centralized location for creating and fetching network trees for clusters.
//go:generate mockgen-wrapper
type Manager interface {
	CreateNetworkTree(clusterID string) tree.NetworkTree
	GetNetworkTree(clusterID string) tree.NetworkTree
	GetReadOnlyNetworkTree(clusterID string) tree.ReadOnlyNetworkTree
	DeleteNetworkTree(clusterID string)
}

func newManager() Manager {
	return &managerImpl{
		trees: make(map[string]tree.NetworkTree),
	}
}
