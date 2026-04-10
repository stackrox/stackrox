package lock

import (
	"context"

	"github.com/stackrox/rox/central/dblock"
	"github.com/stackrox/rox/pkg/postgres"
)

const (
	// migrationAdvisoryLockID is a unique identifier for the migration advisory lock.
	// This value is arbitrary but must be consistent across all Central instances.
	migrationAdvisoryLockID int64 = 7_517_845_236_103_920_641
)

// TryAcquireMigrationLock attempts to acquire the migration advisory lock without blocking.
// Returns whether the lock was acquired, a release function (nil if not acquired), and any error.
func TryAcquireMigrationLock(ctx context.Context, pool postgres.DB) (bool, func(), error) {
	return dblock.TryAcquireAdvisoryLock(ctx, pool, migrationAdvisoryLockID)
}
