package vuln

import (
	"context"
	"slices"

	"github.com/quay/claircore"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/datastore/postgres"
)

// distManager manages updates to known-distributions.
type distManager struct {
	store postgres.MatcherStore

	// mutex protects access to known.
	mutex sync.RWMutex
	// known tracks all known distributions.
	known []claircore.Distribution
}

func newDistManager(store postgres.MatcherStore) *distManager {
	return &distManager{
		store: store,
	}
}

// update updates the currently known distributions, on-demand.
//
// It is *not* safe to call update concurrently.
func (dm *distManager) update(ctx context.Context) (err error) {
	ctx = zlog.ContextWithValues(ctx, "component", "updater/vuln/dists.update")
	zlog.Info(ctx).Msg("updating vuln distributions")
	defer func() {
		zlog.Info(ctx).Bool("updated", err == nil).Msg("done updating vuln distributions")
	}()

	var dists []claircore.Distribution
	dists, err = dm.store.Distributions(ctx)
	if err != nil {
		return
	}

	func() {
		dm.mutex.Lock()
		defer dm.mutex.Unlock()

		dm.known = nil // Hint to GC.
		dm.known = dists
	}()

	return
}

// get returns the currently known distributions.
func (dm *distManager) get() []claircore.Distribution {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	known := slices.Clone(dm.known)
	return known
}
