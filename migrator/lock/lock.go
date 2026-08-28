package lock

import (
	"context"

	"github.com/stackrox/rox/pkg/dblock"
	"github.com/stackrox/rox/pkg/postgres"
)

// TryAcquireMigrationLock attempts to acquire the migration advisory lock without blocking.
// Returns whether the lock was acquired, a release function (nil if not acquired), and any error.
func TryAcquireMigrationLock(ctx context.Context, pool postgres.DB) (bool, func(), error) {
	return dblock.TryAcquireAdvisoryLock(ctx, pool, dblock.MigrationLockID)
}
