package networktree

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
)

// Manager provides a centralized location for creating and fetching network trees for clusters.
//
//go:generate mockgen-wrapper
type Manager interface {
	Initialize(entitiesByCluster map[string][]*storage.NetworkEntityInfo) error
	CreateDefaultNetworkTree(ctx context.Context) tree.NetworkTree
	CreateNetworkTree(ctx context.Context, clusterID string) tree.NetworkTree
	GetNetworkTree(ctx context.Context, clusterID string) tree.NetworkTree
	GetReadOnlyNetworkTree(ctx context.Context, clusterID string) tree.ReadOnlyNetworkTree
	GetDefaultNetworkTree(ctx context.Context) tree.ReadOnlyNetworkTree
	DeleteNetworkTree(ctx context.Context, clusterID string)
}

func newManager() Manager {
	return &managerImpl{
		trees:          make(map[string]tree.NetworkTree),
		initializedSig: concurrency.NewSignal(),
	}
}
