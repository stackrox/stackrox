package manifest

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/datastore/postgres"
)

const (
	initName = `manifest-init`

	gcName     = `manifest-garbage-collection`
	gcInterval = 6 * time.Hour
)

var (
	jitterHours = []time.Duration{
		-1 * time.Hour,
		0 * time.Hour,
		1 * time.Hour,
	}
)

type GC struct {
	ctx    context.Context
	cancel context.CancelFunc

	metadataStore postgres.IndexerMetadataStore
	locker        updates.LockSource
	deleteFunc    DeleteManifestsFunc
}

type DeleteManifestsFunc func(ctx context.Context, d ...claircore.Digest) ([]claircore.Digest, error)

func NewGC(ctx context.Context, metadataStore postgres.IndexerMetadataStore, locker updates.LockSource, deleteFunc DeleteManifestsFunc) *GC {
	ctx, cancel := context.WithCancel(ctx)
	return &GC{
		ctx:    ctx,
		cancel: cancel,

		metadataStore: metadataStore,
		locker:        locker,
		deleteFunc:    deleteFunc,
	}
}

func (g *GC) Start() error {
	ctx := zlog.ContextWithValues(g.ctx, "component", "indexer/manifest/GC.Start")

	if err := g.initGC(); err != nil {
		zlog.Error(ctx).Err(err).Msg("errors encountered during manifest GC initialization")
	}
	if err := g.runGC(); err != nil {
		zlog.Error(ctx).Err(err).Msg("errors encountered during manifest GC run")
	}

	t := time.NewTimer(gcInterval + jitter())
	defer t.Stop()
	for {
		select {
		case <-g.ctx.Done():
			return g.ctx.Err()
		case <-t.C:
			if err := g.runGC(); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during manifest GC run")
			}

			t.Reset(gcInterval + jitter())
		}
	}
}

func (g *GC) initGC() error {
	// Use TryLock instead of Lock in case an initialization is already happening.
	// There is no need to run another one.
	ctx, done := g.locker.TryLock(g.ctx, initName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("lock context canceled, garbage collection initialization already running")
		return nil
	}

	ms, err := g.metadataStore.MigrateManifests(g.ctx)
	if err != nil {
		return err
	}
	if len(ms) > 0 {
		zlog.Debug(ctx).Strs("migrated_manifests", ms).Msg("migrated missing manifest metadata")
	}
	zlog.Info(ctx).Int("migrated_manifests", len(ms)).Msg("migrated missing manifest metadata")

	return nil
}

func (g *GC) runGC() error {
	// Use TryLock instead of Lock in case a GC cycle is already happening.
	// No need to run simultaneous GC operations.
	ctx, done := g.locker.TryLock(g.ctx, gcName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("lock context canceled, garbage collection already running")
		return nil
	}

	ms, err := g.metadataStore.GCManifests(g.ctx, time.Now())
	if err != nil {
		return err
	}
	if len(ms) > 0 {
		zlog.Debug(ctx).Strs("deleted_manifests", ms).Msg("deleted expired manifest metadata")
	}
	zlog.Info(ctx).Int("deleted_manifests", len(ms)).Msg("deleted expired manifest metadata")

	return nil
}

func (g *GC) Stop() error {
	g.cancel()
	return nil
}

func jitter() time.Duration {
	return jitterHours[rand.IntN(len(jitterHours))]
}
