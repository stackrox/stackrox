package distribution

import (
	"context"
	"math/rand"
	"slices"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/datastore/postgres"
)

const updateInterval = 24 * time.Hour

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	jitterHours = []time.Duration{
		1 * time.Hour,
		2 * time.Hour,
		3 * time.Hour,
		4 * time.Hour,
		5 * time.Hour,
	}
)

// Updater represents a known-distribution updater.
// An Updater reaches out to the DB periodically to fetch the currently known distributions.
type Updater struct {
	ctx    context.Context
	cancel context.CancelFunc

	store postgres.MatcherStore

	// known tracks all known distributions.
	known []claircore.Distribution
	// mutex protects access to known.
	mutex sync.RWMutex

	// vulnInitializedFunc returns true if vulnerabilities are ready, so we can fetch
	// distributions, if nil no checks are done.
	vulnInitializedFunc func(ctx context.Context) bool
}

// New creates a new Updater.
func New(ctx context.Context, store postgres.MatcherStore, vulnInitializedFunc func(ctx context.Context) bool) (*Updater, error) {
	ctx, cancel := context.WithCancel(ctx)
	u := &Updater{
		ctx:    ctx,
		cancel: cancel,

		store: store,

		vulnInitializedFunc: vulnInitializedFunc,
	}
	return u, nil
}

// Known returns the currently known distributions.
func (u *Updater) Known() []claircore.Distribution {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	known := slices.Clone(u.known)
	return known
}

// Start begins the update proces.
func (u *Updater) Start() error {
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/distribution/Updater.Start")
	// Schedule the initial update to occur randomly anywhere from now up to the
	// specified limit minutes, to reduce chances of simultaneous runs.
	d := time.Duration(rand.Float64() * float64(1*time.Minute))
	zlog.Info(ctx).Msgf("initial update in %s", d)
	t := time.NewTimer(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			// Concurrent reads are safe because this goroutine is the only writer.
			isInitial := u.known == nil
			if isInitial {
				if u.vulnInitializedFunc != nil && !u.vulnInitializedFunc(u.ctx) {
					d := 15 * time.Minute
					zlog.Info(ctx).Msgf("vulnerabilities uninitialized: retrying in %s...", d)
					t.Reset(d)
					continue
				}
				zlog.Info(ctx).Msg("starting initial update")
			} else {
				zlog.Info(ctx).Msg("starting update")
			}
			if err := u.update(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
			} else if isInitial {
				zlog.Info(ctx).Msg("completed initial update")
			} else {
				zlog.Info(ctx).Msg("completed update")
			}
			t.Reset(updateInterval + jitter())
		}
	}
}

func (u *Updater) update(ctx context.Context) error {
	dists, err := u.store.Distributions(ctx)
	if err != nil {
		return err
	}

	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.known = nil // Hint to GC.
	u.known = dists

	return nil
}

// Stop stops the update process.
func (u *Updater) Stop() error {
	u.cancel()
	return nil
}

func jitter() time.Duration {
	return jitterHours[random.Intn(len(jitterHours))]
}
