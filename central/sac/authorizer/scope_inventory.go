package authorizer

import (
	"context"
	"time"

	"github.com/pkg/errors"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sync"
)

const scopeInventoryLifetime = 5 * time.Second

// Keep the inventory used by the in-memory scope evaluator separate from ordinary
// store reads. This is a compatibility path until scope evaluation uses SQL.
var scopeInventory = &scopeInventoryCache{load: loadScopeInventory}

// Inventories are immutable once published and shared by request-local checkers.
type inventory struct {
	clusters   []effectiveaccessscope.Cluster
	namespaces []effectiveaccessscope.Namespace
}

type inventoryRefresh struct {
	done      chan struct{}
	inventory inventory
	err       error
}

type scopeInventoryCache struct {
	mutex       sync.Mutex
	inventory   inventory
	refreshedAt time.Time
	refresh     *inventoryRefresh
	load        func(context.Context) (inventory, error)
}

func (c *scopeInventoryCache) get(ctx context.Context) (inventory, error) {
	if err := ctx.Err(); err != nil {
		return inventory{}, err
	}
	result, refresh, ownsRefresh := concurrency.WithLock3(&c.mutex, func() (inventory, *inventoryRefresh, bool) {
		if time.Since(c.refreshedAt) < scopeInventoryLifetime {
			return c.inventory, nil, false
		}
		if c.refresh != nil {
			return inventory{}, c.refresh, false
		}
		c.refresh = &inventoryRefresh{done: make(chan struct{})}
		return inventory{}, c.refresh, true
	})
	if refresh == nil {
		return result, nil
	}
	if !ownsRefresh {
		select {
		case <-ctx.Done():
			return inventory{}, ctx.Err()
		case <-refresh.done:
			return refresh.inventory, refresh.err
		}
	}
	result, err := c.load(ctx)
	concurrency.WithLock(&c.mutex, func() {
		if err == nil {
			c.inventory = result
			c.refreshedAt = time.Now()
			refresh.inventory = result
		}
		refresh.err = err
		c.refresh = nil
		close(refresh.done)
	})
	return refresh.inventory, err
}

func loadScopeInventory(ctx context.Context) (inventory, error) {
	clusters, err := clusterStore.Singleton().GetClustersForSAC(ctx)
	if err != nil {
		return inventory{}, errors.Wrap(err, "reading all clusters")
	}
	namespaces, err := namespaceStore.Singleton().GetNamespacesForSAC(ctx)
	if err != nil {
		return inventory{}, errors.Wrap(err, "reading all namespaces")
	}
	return inventory{clusters: clusters, namespaces: namespaces}, nil
}
