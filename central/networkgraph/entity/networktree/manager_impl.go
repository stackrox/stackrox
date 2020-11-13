package networktree

import (
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	defaultNetworksClusterID = ""
)

type managerImpl struct {
	trees map[string]tree.NetworkTree

	lock sync.RWMutex
}

func (f *managerImpl) GetNetworkTree(clusterID string) tree.NetworkTree {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.trees[clusterID]
}

func (f *managerImpl) GetReadOnlyNetworkTree(clusterID string) tree.ReadOnlyNetworkTree {
	return f.GetNetworkTree(clusterID)
}

func (f *managerImpl) GetDefaultNetworkTree() tree.ReadOnlyNetworkTree {
	return f.GetNetworkTree(defaultNetworksClusterID)
}

func (f *managerImpl) CreateNetworkTree(clusterID string) tree.NetworkTree {
	f.lock.Lock()
	defer f.lock.Unlock()

	t := f.trees[clusterID]
	if t != nil {
		return t
	}

	t = tree.NewDefaultNetworkTreeWrapper()
	f.trees[clusterID] = t
	return t
}

func (f *managerImpl) DeleteNetworkTree(clusterID string) {
	f.lock.Lock()
	defer f.lock.Unlock()

	delete(f.trees, clusterID)
}
