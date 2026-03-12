package lock

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/log"
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
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return false, nil, errors.Wrap(err, "acquiring connection for migration lock")
	}

	var acquired bool
	err = conn.PgxPoolConn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", migrationAdvisoryLockID).Scan(&acquired)
	if err != nil {
		conn.Release()
		return false, nil, errors.Wrap(err, "trying migration advisory lock")
	}

	if !acquired {
		conn.Release()
		return false, nil, nil
	}

	log.WriteToStderr("Migration advisory lock acquired.")
	return true, makeRelease(conn), nil
}

func makeRelease(conn *postgres.Conn) func() {
	released := false
	return func() {
		if released {
			return
		}
		released = true
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer unlockCancel()
		_, err := conn.PgxPoolConn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationAdvisoryLockID)
		if err != nil {
			log.WriteToStderrf("Warning: failed to release migration advisory lock: %v", err)
		} else {
			log.WriteToStderr("Migration advisory lock released.")
		}
		conn.Release()
	}
}
