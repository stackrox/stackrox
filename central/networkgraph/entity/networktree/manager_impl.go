package networktree

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	defaultNetworksClusterID = ""
	errWaitOnInit            = "context error waiting for network tree manager init"
)

var (
	log = logging.LoggerForModule()
)

type managerImpl struct {
	trees map[string]tree.NetworkTree

	initializedSig concurrency.Signal
	lock           sync.RWMutex
}

func (f *managerImpl) Initialize(entitiesByCluster map[string][]*storage.NetworkEntityInfo) error {
	defer f.initializedSig.Signal()

	for cluster, entities := range entitiesByCluster {
		if _, err := f.createNoLock(cluster, entities...); err != nil {
			return err
		}
	}
	return nil
}

func (f *managerImpl) GetNetworkTree(ctx context.Context, clusterID string) tree.NetworkTree {
	if !f.initialized(ctx) {
		log.Error(errors.Wrap(ctx.Err(), errWaitOnInit))
		return nil
	}

	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.trees[clusterID]
}

func (f *managerImpl) GetReadOnlyNetworkTree(ctx context.Context, clusterID string) tree.ReadOnlyNetworkTree {
	return f.GetNetworkTree(ctx, clusterID)
}

func (f *managerImpl) GetDefaultNetworkTree(ctx context.Context) tree.ReadOnlyNetworkTree {
	return f.GetNetworkTree(ctx, defaultNetworksClusterID)
}

func (f *managerImpl) CreateDefaultNetworkTree(ctx context.Context) tree.NetworkTree {
	return f.CreateNetworkTree(ctx, defaultNetworksClusterID)
}

func (f *managerImpl) CreateNetworkTree(ctx context.Context, clusterID string) tree.NetworkTree {
	if !f.initialized(ctx) {
		return nil
	}

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

func (f *managerImpl) createNoLock(clusterID string, entities ...*storage.NetworkEntityInfo) (tree.NetworkTree, error) {
	t := f.trees[clusterID]
	// Fail only if there are entities to insert.
	if t != nil {
		if len(entities) > 0 {
			return nil, errors.Errorf("network tree for cluster %q already exists", clusterID)
		}
		return t, nil
	}

	t, err := tree.NewNetworkTreeWrapper(entities)
	if err != nil {
		return nil, errors.Wrapf(err, "creating network tree for cluster %q", clusterID)
	}
	f.trees[clusterID] = t
	return t, nil
}

func (f *managerImpl) DeleteNetworkTree(ctx context.Context, clusterID string) {
	if !f.initialized(ctx) {
		log.Error(errors.Wrap(ctx.Err(), errWaitOnInit))
		return
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	delete(f.trees, clusterID)
}

func (f *managerImpl) initialized(ctx context.Context) bool {
	return concurrency.WaitInContext(&f.initializedSig, ctx)
}
