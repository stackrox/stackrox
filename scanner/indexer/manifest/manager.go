package manifest

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/datastore/postgres"
)

const (
	// gcName is the name of the GC process.
	// The locker uses this to prevent concurrent GC runs.
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
	metadataStore postgres.IndexerMetadataStore
	locker        updates.LockSource

	// gcThrottle specifies the number of manifests to delete during a non-full GC run.
	gcThrottle int
	// interval specifies the amount of time between normal GC runs.
	interval time.Duration
	// fullInterval specifies the amount of time between full GC runs.
	fullInterval time.Duration

	// cancelGC is a signal to cancel all in-progress GC runs.
	cancelGC chan struct{}
}

// NewManager creates a manifest manager.
func NewManager(ctx context.Context, metadataStore postgres.IndexerMetadataStore, locker updates.LockSource) *Manager {
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
		metadataStore: metadataStore,
		locker:        locker,

		gcThrottle:   gcThrottle,
		interval:     interval,
		fullInterval: fullInterval,

		cancelGC: make(chan struct{}, 1),
	}
}

// StartGC attempts to:
//  1. Acquire a global lock via the provided locker such that only a single Manager in a distributed system runs for a
//     metadataStore.
//  2. Migrate all known manifests not yet part of the garbage collection process, then run a full garbage collection.
//     This process will be retried if not initially successful. Doesn't run after the first successful run for a
//     particular Manager, but other Managers in a distributed system will re-run the migration.
//  3. Begin periodic garbage collection with throttles.
//  4. Begin periodic full garbage collection.
//
// The global lock will be acquired before attempting any other work. Additionally, if there are any reasons the lock
// is released, e.g., context cancellation, network failure, etc, the rest of the garbage collection process is notified
// and stopped.
//
// Note that the entire process (migrating manifests, full garbage collection, etc.) will happen each time a new
// Manager acquires a lock on the database.
func (m *Manager) StartGC(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "indexer/manifest/Manager.StartGC")

	// If we don't acquire the lock during the initial attempt, we want to run
	// the migration and the full garbage collection during one of the interval
	// attempts. But we only want to either run the migration or whichever
	// garbage collection method relevant to that particular interval, so we
	// set up the migration process to ensure it only runs once.
	migrationDone := false
	migrateOnce := sync.OnceValue(func() error {
		// Set any manifests indexed prior to the existence of the manifest_metadata table
		// to expire immediately.
		// TODO(ROX-26957): Consider moving this elsewhere so we do not block initialization.
		// TODO(ROX-26995): Consider updating the immediate purge condition.
		//  It may be possible we want to purge all manifests upon startup for other reasons.
		err := m.migrateManifests(ctx, time.Now())
		if err != nil {
			// TODO(ROX-26958): Consider just logging this instead once we start deleting entries
			//  missing from the metadata table, too.
			return fmt.Errorf("migrating manifests to metadata store: %w", err)
		}
		zlog.Debug(ctx).Msg("migrated manifests")

		if err := m.fullGC(ctx); err != nil {
			zlog.Error(ctx).Err(err).Msg("errors encountered during initial full manifest GC run")
		}
		zlog.Debug(ctx).Msg("full GC after migration completed")

		zlog.Info(ctx).Msg("migration process completed successfully")
		migrationDone = true
		return nil
	})

	// Start the lock coordination.
	var lockCtx context.Context
	var lockReleaseFunc context.CancelFunc

	zlog.Debug(ctx).Msg("attempting initial lock acquisition")
	lockCtx, lockReleaseFunc = m.locker.TryLock(ctx, gcName)
	defer lockReleaseFunc()
	if err := lockCtx.Err(); err != nil {
		zlog.Info(ctx).Err(err).Msg("did not acquire lock, another manager likely has it")
		lockReleaseFunc()
	} else {
		zlog.Info(ctx).Msg("lock acquired")
		if err := migrateOnce(); err != nil {
			return err
		}
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
		case <-m.cancelGC:
			return nil
		// Lost the lock. Immediately try to reacquire the lock.
		case <-lockCtx.Done():
			zlog.Warn(ctx).Msg("lock lost, retrying")
			lockReleaseFunc()
			lockCtx, lockReleaseFunc = m.locker.TryLock(ctx, gcName)
			if err := lockCtx.Err(); err != nil {
				zlog.Info(ctx).Err(err).Msg("did not acquire lock")
				lockReleaseFunc()
			} else {
				zlog.Info(ctx).Msg("lock acquired")
			}
		case <-t.C:
			// Check if we have the lock.
			if lockCtx.Err() != nil {
				zlog.Debug(ctx).Msg("attempting to acquire lock during interval")
				lockCtx, lockReleaseFunc = m.locker.TryLock(ctx, gcName)
				if err := lockCtx.Err(); err != nil {
					zlog.Info(ctx).Err(err).Msg("did not acquire lock, another manager likely has it")
					lockReleaseFunc()
					// Failed to acquire the lock. Skip GC and reset the timer.
					goto resetInterval
				}

				zlog.Info(ctx).Msg("lock maintained")
			}

			if !migrationDone {
				if err := migrateOnce(); err != nil {
					return err
				}
			} else {
				if err := m.partialGCWithThrottle(ctx); err != nil {
					zlog.Error(ctx).Err(err).Msg("errors encountered during manifest GC run")
				}
			}

		resetInterval:
			interval = m.interval + jitter()
			t.Reset(interval)
			zlog.Info(ctx).Msgf("next manifest metadata GC attempt will be in about %v", interval)
		case <-tFull.C:
			// Check if we have the lock.
			if lockCtx.Err() != nil {
				zlog.Debug(ctx).Msg("attempting to acquire lock during full interval")
				lockCtx, lockReleaseFunc = m.locker.TryLock(ctx, gcName)
				if err := lockCtx.Err(); err != nil {
					zlog.Info(ctx).Err(err).Msg("did not acquire lock, another manager likely has it")
					lockReleaseFunc()
					// Failed to acquire the lock. Skip GC and reset the timer.
					goto resetFullInterval
				}

				zlog.Info(ctx).Msg("lock maintained")
			}

			if !migrationDone {
				if err := migrateOnce(); err != nil {
					return err
				}
			} else {
				if err := m.fullGC(ctx); err != nil {
					zlog.Error(ctx).Err(err).Msg("errors encountered during manifest GC run")
				}
			}

		resetFullInterval:
			fullInterval = m.fullInterval + jitter()
			tFull.Reset(fullInterval)
			zlog.Info(ctx).Msgf("next full manifest metadata GC attempt will be in about %v", fullInterval)
		}
	}
}

// migrateManifests migrates known claircore manifests into the manifest_metadata table, so the Manager can garbage
// collection them later. Unless there's a good reason, it's strongly encouraged to call runFullGC after
// migrateManifests.
func (m *Manager) migrateManifests(ctx context.Context, expiration time.Time) error {
	ctx = zlog.ContextWithValues(ctx, "component", "indexer/manifest/Manager.migrateManifests")

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

// runFullGC runs the garbage collection process without the throttle mechanism. Mostly intended to run on startup, so
// the manager has a fresh slate to work with when running the garbage collection process with the throttle mechanism.
// Assumes the global lock has been acquired.
func (m *Manager) fullGC(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "indexer/manifest/Manager.fullGC")

	zlog.Info(ctx).Msg("starting manifest metadata garbage collection")

	var ms []string
	// Set i to any int greater than 0 to start the loop.
	i := 1
	for i > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			deleted, err := m.metadataStore.GCManifests(ctx, time.Now(), postgres.WithGCThrottle(m.gcThrottle))
			if err != nil {
				return err
			}
			i = len(deleted)
			ms = append(ms, deleted...)
		}
	}

	if len(ms) > 0 {
		zlog.Debug(ctx).Strs("deleted_manifest_metadata", ms).Msg("deleted expired manifest metadata")
	}
	zlog.Info(ctx).Int("deleted_manifest_metadata", len(ms)).Msg("deleted expired manifest metadata")

	return nil
}

// partialGCWithThrottle runs the garbage collection process with the throttle mechanism configured via gcThrottle.
// Assumes the global lock has been acquired.
func (m *Manager) partialGCWithThrottle(ctx context.Context) error {
	ctx = zlog.ContextWithValues(ctx, "component", "indexer/manifest/Manager.partialGCWithThrottle")

	zlog.Info(ctx).Msg("starting manifest metadata garbage collection")

	ms, err := m.metadataStore.GCManifests(ctx, time.Now(), postgres.WithGCThrottle(m.gcThrottle))
	if err != nil {
		return err
	}

	if len(ms) > 0 {
		zlog.Debug(ctx).Strs("deleted_manifest_metadata", ms).Msg("deleted expired manifest metadata")
	}
	zlog.Info(ctx).Int("deleted_manifest_metadata", len(ms)).Msg("deleted expired manifest metadata")

	return nil
}

// StopGC ends periodic garbage collection and releases the global lock.
func (m *Manager) StopGC() error {
	m.cancelGC <- struct{}{}
	return nil
}

func jitter() time.Duration {
	return jitterMinutes[rand.IntN(len(jitterMinutes))]
}
