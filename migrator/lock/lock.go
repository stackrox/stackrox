package lock

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/postgres"
)

// TryAcquireMigrationLock attempts to acquire the migration advisory lock without blocking.
// Returns whether the lock was acquired, a release function (nil if not acquired), and any error.
func TryAcquireMigrationLock(ctx context.Context, pool postgres.DB) (bool, func(), error) {
	return dblock.TryAcquireAdvisoryLock(ctx, pool, dblock.MigrationLockID)
}

// AcquireMigrationLock waits for the migration lock until ctx is canceled.
// The context only limits acquisition, not how long the caller holds the lock.
func AcquireMigrationLock(ctx context.Context, pool postgres.DB) (func(), error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		acquired, release, err := TryAcquireMigrationLock(ctx, pool)
		if err != nil {
			return nil, err
		}
		if acquired {
			return release, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
