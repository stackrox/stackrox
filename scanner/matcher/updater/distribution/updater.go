package distribution

import (
	"context"
	"math/rand"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"golang.org/x/exp/slices"
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

type Updater struct {
	ctx    context.Context
	cancel context.CancelFunc

	store postgres.MatcherStore

	// known tracks all known distributions.
	known []claircore.Distribution
	// mutex protects access to known.
	mutex sync.RWMutex
}

func New(ctx context.Context, store postgres.MatcherStore) (*Updater, error) {
	ctx, cancel := context.WithCancel(ctx)
	u := &Updater{
		ctx:    ctx,
		cancel: cancel,

		store: store,
	}
	return u, nil
}

func (u *Updater) Known() []claircore.Distribution {
	known := func() []claircore.Distribution {
		u.mutex.RLock()
		defer u.mutex.RUnlock()

		return slices.Clone(u.known)
	}()

	return known
}

func (u *Updater) Start() error {
	ctx := zlog.ContextWithValues(u.ctx, "component", "matcher/updater/distribution/Updater.Start")

	zlog.Info(ctx).Msg("starting initial update")
	if err := u.update(ctx); err != nil {
		zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
	}
	zlog.Info(ctx).Msg("completed initial update")

	t := time.NewTimer(updateInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			zlog.Info(ctx).Msg("starting update")
			if err := u.update(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during updater run")
			}
			zlog.Info(ctx).Msg("completed update")

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

func (u *Updater) Stop() error {
	u.cancel()
	return nil
}

func jitter() time.Duration {
	return jitterHours[random.Intn(len(jitterHours))]
}
