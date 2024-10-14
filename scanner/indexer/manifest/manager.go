package manifest

import (
	"context"
	"time"

	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/scanner/datastore/postgres"
)

const (
	// migrateName is the name of the manifest migration process.
	// This is used by the lock to prevent concurrent initializations.
	migrateName = `manifest-migrate`

	// gcName is the name of the GC process.
	// This is used by the lock to prevent concurrent GC runs.
	gcName = `manifest-garbage-collection`

	// minGCInterval specifies the minimum interval between GC runs.
	minGCInterval = time.Hour
)

// Manager represents an indexer manifest manager.
//
// After initialization, it periodically runs a process
// to identify expired manifests and delete them from
// both the manifest metadata storage maintained by StackRox
// and the manifest storage maintained by ClairCore.
type Manager struct {
	ctx      context.Context
	gcCtx    context.Context
	gcCancel context.CancelFunc

	metadataStore postgres.IndexerMetadataStore
	locker        updates.LockSource

	// interval specifies the amount of time between GC runs.
	interval time.Duration
}

// NewManager creates a manifest manager.
func NewManager(ctx context.Context, metadataStore postgres.IndexerMetadataStore, locker updates.LockSource) *Manager {
	gcCtx, gcCancel := context.WithCancel(ctx)
	interval := env.ScannerV4ManifestGCInterval.DurationSetting()
	if interval < minGCInterval {
		zlog.Info(ctx).Msgf("configured manifest GC interval is too small: setting to %v", minGCInterval)
		interval = minGCInterval
	}
	return &Manager{
		ctx:      ctx,
		gcCtx:    gcCtx,
		gcCancel: gcCancel,

		metadataStore: metadataStore,
		locker:        locker,

		interval: interval,
	}
}

// MigrateManifests migrates manifests into the manifest_metadata table.
func (m *Manager) MigrateManifests() error {
	// Use TryLock instead of Lock in case a migration is already happening.
	// There is no need to run another one.
	ctx, done := m.locker.TryLock(m.ctx, migrateName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("lock context canceled, manifest migration already running")
		return nil
	}

	ms, err := m.metadataStore.MigrateManifests(ctx)
	if err != nil {
		return err
	}
	if len(ms) > 0 {
		zlog.Debug(ctx).Strs("migrated_manifests", ms).Msg("migrated missing manifest metadata")
	}
	zlog.Info(ctx).Int("migrated_manifests", len(ms)).Msg("migrated missing manifest metadata")

	return nil
}

// StartGC begins periodic garbage collection.
func (m *Manager) StartGC() error {
	ctx := zlog.ContextWithValues(m.gcCtx, "component", "indexer/manifest/GC.Start")

	if err := m.runGC(ctx); err != nil {
		zlog.Error(ctx).Err(err).Msg("errors encountered during manifest GC run")
	}
	zlog.Info(ctx).Msgf("next manifest metadata GC run will be in about %v", m.interval)

	t := time.NewTimer(m.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := m.runGC(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during manifest GC run")
			}

			t.Reset(m.interval)
			zlog.Info(ctx).Msgf("next manifest metadata GC run will be in about %v", m.interval)
		}
	}
}

func (m *Manager) runGC(ctx context.Context) error {
	// Use TryLock instead of Lock in case a GC cycle is already happening.
	// No need to run simultaneous GC operations.
	ctx, done := m.locker.TryLock(ctx, gcName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("lock context canceled, garbage collection already running")
		return nil
	}

	zlog.Info(ctx).Msg("starting manifest metadata garbage collection")
	ms, err := m.metadataStore.GCManifests(ctx, time.Now())
	if err != nil {
		return err
	}
	if len(ms) > 0 {
		zlog.Debug(ctx).Strs("deleted_manifests", ms).Msg("deleted expired manifest metadata")
	}
	zlog.Info(ctx).Int("deleted_manifests", len(ms)).Msg("deleted expired manifest metadata")

	return nil
}

// StopGC ends periodic garbage collection.
func (m *Manager) StopGC() error {
	m.gcCancel()
	return nil
}
