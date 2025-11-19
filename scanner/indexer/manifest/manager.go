package manifest

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/scanner/datastore/postgres"
)

const (
	// migrateName is the name of the manifest migration process.
	// This is used by the lock to prevent concurrent migrations.
	migrateName = `manifest-migrate`

	// gcName is the name of the GC process.
	// This is used by the lock to prevent concurrent GC runs.
	gcName = `manifest-garbage-collection`

	// minGCThrottle specifies the minimum number of manifests to GC per run.
	minGCThrottle = 1
)

var (
	// minGCInterval specifies the minimum interval between GC runs.
	minGCInterval = minGCIntervalDuration()
	// minFullGCInterval specifies the minimum interval between full GC runs.
	minFullGCInterval = minFullGCIntervalDuration()

	jitterMinutes = []time.Duration{
		-10 * time.Minute,
		-5 * time.Minute,
		0 * time.Minute,
		5 * time.Minute,
		10 * time.Minute,
	}
)

// minGCIntervalDuration returns the minimum GC interval duration.
// For release builds: 1 hour
// For dev builds: 1 minute
func minGCIntervalDuration() time.Duration {
	if buildinfo.ReleaseBuild {
		return time.Hour
	}
	return time.Minute
}

// minFullGCIntervalDuration returns the minimum GC interval duration.
// For release builds: 1 hour
// For dev builds: 1 minute
func minFullGCIntervalDuration() time.Duration {
	if buildinfo.ReleaseBuild {
		return time.Hour
	}
	return time.Minute
}

// Manager represents an indexer manifest manager.
//
// After initialization, it periodically runs a process
// to identify expired manifests and delete them from
// both the manifest metadata storage maintained by StackRox
// and the manifest storage maintained by Claircore.
type Manager struct {
	gcCtx    context.Context
	gcCancel context.CancelFunc

	metadataStore      postgres.IndexerMetadataStore
	externalIndexStore postgres.ExternalIndexStore
	locker             updates.LockSource

	// gcThrottle specifies the number of manifests to delete during a non-full GC run.
	gcThrottle int
	// interval specifies the amount of time between normal GC runs.
	interval time.Duration
	// fullInterval specifies the amount of time between full GC runs.
	fullInterval time.Duration
}

// NewManager creates a manifest manager.
func NewManager(ctx context.Context, metadataStore postgres.IndexerMetadataStore, externalIndexStore postgres.ExternalIndexStore, locker updates.LockSource) *Manager {
	gcCtx, gcCancel := context.WithCancel(ctx)

	interval := env.ScannerV4ManifestGCInterval.DurationSetting()
	if interval < minGCInterval {
		zlog.Warn(ctx).Msgf("configured manifest GC interval (%v) is too small: setting to %v", interval, minGCInterval)
		interval = minGCInterval
	}

	fullInterval := env.ScannerV4FullManifestGCInterval.DurationSetting()
	if fullInterval < minFullGCInterval {
		zlog.Warn(ctx).Msgf("configured full manifest GC interval (%v) is too small: setting to %v", fullInterval, minFullGCInterval)
		fullInterval = minFullGCInterval
	}

	gcThrottle := env.ScannerV4ManifestGCThrottle.IntegerSetting()
	if gcThrottle < minGCThrottle {
		zlog.Warn(ctx).Msgf("configured manifest GC throttle (%d) is too small: setting to %d", gcThrottle, minGCThrottle)
		gcThrottle = minGCThrottle
	}

	return &Manager{
		gcCtx:    gcCtx,
		gcCancel: gcCancel,

		metadataStore:      metadataStore,
		externalIndexStore: externalIndexStore,
		locker:             locker,

		gcThrottle:   gcThrottle,
		interval:     interval,
		fullInterval: fullInterval,
	}
}

// MigrateManifests migrates manifests into the manifest_metadata table.
func (m *Manager) MigrateManifests(ctx context.Context, expiration time.Time) error {
	ctx = zlog.ContextWithValues(ctx, "component", "indexer/manifest/Manager.MigrateManifests")

	// Use TryLock instead of Lock in case a migration is already happening.
	// There is no need to run another one.
	ctx, done := m.locker.TryLock(ctx, migrateName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("lock context canceled, manifest migration already running")
		return nil
	}

	ms, err := m.metadataStore.MigrateManifests(ctx, expiration)
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
	ctx := zlog.ContextWithValues(m.gcCtx, "component", "indexer/manifest/Manager.StartGC")

	if err := m.runFullGC(ctx); err != nil {
		zlog.Error(ctx).Err(err).Msg("errors encountered during initial full manifest GC run")
	}

	interval := m.interval + jitter()
	zlog.Info(ctx).Msgf("next manifest metadata GC run will be in about %v", interval)
	t := time.NewTimer(interval)
	defer t.Stop()

	fullInterval := m.fullInterval + jitter()
	zlog.Info(ctx).Msgf("next full manifest metadata GC run will be in about %v", fullInterval)
	tFull := time.NewTimer(fullInterval)
	defer tFull.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := m.runGC(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during manifest GC run")
			}

			interval = m.interval + jitter()
			t.Reset(interval)
			zlog.Info(ctx).Msgf("next manifest metadata GC run will be in about %v", interval)
		case <-tFull.C:
			if err := m.runFullGC(ctx); err != nil {
				zlog.Error(ctx).Err(err).Msg("errors encountered during full manifest GC run")
			}

			fullInterval = m.fullInterval + jitter()
			tFull.Reset(fullInterval)
			zlog.Info(ctx).Msgf("next full manifest metadata GC run will be in about %v", fullInterval)
		}
	}
}

func (m *Manager) runFullGC(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "indexer/manifest/Manager.runFullGC")

	// Use Lock instead of TryLock to ensure we get the lock
	// and run a full GC.
	ctx, done := m.locker.Lock(ctx, gcName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Warn(ctx).Err(err).Msg("lock context canceled")
		return err
	}

	zlog.Info(ctx).Msg("starting manifest metadata garbage collection")

	var ms []string
	var irs []string
	// Set i to any int greater than 0 to start the loop.
	i := 1
	for i > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			deletedManifests, gcManifestsErr := m.runGCManifestsNoLock(ctx)
			deleteIndexReports, gcIndexReportsErr := m.runGCIndexReportsNoLock(ctx)
			err := errors.Join(gcManifestsErr, gcIndexReportsErr)
			if err != nil {
				return err
			}
			i = len(deletedManifests) + len(deleteIndexReports)
			ms = append(ms, deletedManifests...)
			irs = append(irs, deleteIndexReports...)
		}
	}

	if len(ms) > 0 {
		zlog.Debug(ctx).Strs("deleted_manifest_metadata", ms).Msg("deleted expired manifest metadata")
	}
	zlog.Info(ctx).Int("deleted_manifest_metadata", len(ms)).Msg("deleted expired manifest metadata")

	if len(irs) > 0 {
		zlog.Debug(ctx).
			Strs("deleted_external_index_reports", irs).
			Msg("deleted expired external index reports")
	}
	zlog.Info(ctx).
		Int("deleted_external_index_reports", len(irs)).
		Msg("deleted expired external index reports")

	return nil
}

func (m *Manager) runGC(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "indexer/manifest/Manager.runGC")

	// Use TryLock instead of Lock in case a GC cycle is already happening.
	// No need to run simultaneous GC operations.
	ctx, done := m.locker.TryLock(ctx, gcName)
	defer done()
	if err := ctx.Err(); err != nil {
		zlog.Debug(ctx).
			Err(err).
			Msg("lock context canceled, garbage collection already running")
		return err
	}

	zlog.Info(ctx).Msg("starting manifest metadata garbage collection")

	ms, gcManifestsErr := m.runGCManifestsNoLock(ctx)
	irs, gcIndexReportsErr := m.runGCIndexReportsNoLock(ctx)
	err := errors.Join(gcManifestsErr, gcIndexReportsErr)
	if err != nil {
		return err
	}

	if len(ms) > 0 {
		zlog.Debug(ctx).Strs("deleted_manifest_metadata", ms).Msg("deleted expired manifest metadata")
	}
	zlog.Info(ctx).Int("deleted_manifest_metadata", len(ms)).Msg("deleted expired manifest metadata")

	if len(irs) > 0 {
		zlog.Debug(ctx).
			Strs("deleted_external_index_reports", irs).
			Msg("deleted expired external index reports")
	}
	zlog.Info(ctx).
		Int("deleted_external_index_reports", len(irs)).
		Msg("deleted expired external index reports")

	return nil
}

// runGCManifestsNoLock runs the actual garbage collection cycle.
// DO NOT CALL THIS UNLESS THE manifest-garbage-collection LOCK IS ACQUIRED.
func (m *Manager) runGCManifestsNoLock(ctx context.Context) ([]string, error) {
	return m.metadataStore.GCManifests(ctx, time.Now(), postgres.WithGCThrottle(m.gcThrottle))
}

// runGCIndexReportsNoLock runs the actual garbage collection cycle.
// DO NOT CALL THIS UNLESS THE manifest-garbage-collection LOCK IS ACQUIRED.
func (m *Manager) runGCIndexReportsNoLock(ctx context.Context) ([]string, error) {
	return m.externalIndexStore.GCIndexReports(ctx, time.Now(), postgres.WithGCThrottle(m.gcThrottle))
}

// StopGC ends periodic garbage collection.
func (m *Manager) StopGC() error {
	m.gcCancel()
	return nil
}

func jitter() time.Duration {
	return jitterMinutes[rand.IntN(len(jitterMinutes))]
}
