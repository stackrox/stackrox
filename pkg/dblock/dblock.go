package dblock

import (
	"context"

	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// TryAcquireAdvisoryLock attempts to acquire a PostgreSQL advisory lock with the given ID without blocking.
// Returns whether the lock was acquired, a release function (nil if not acquired), and any error.
func TryAcquireAdvisoryLock(ctx context.Context, pool postgres.DB, lockID int64) (bool, func(), error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return false, nil, errors.Wrap(err, "acquiring connection for advisory lock")
	}

	var acquired bool
	err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired)
	if err != nil {
		conn.Release()
		return false, nil, errors.Wrap(err, "trying advisory lock")
	}

	if !acquired {
		conn.Release()
		return false, nil, nil
	}

	log.Infof("Advisory lock %d acquired.", lockID)
	return true, makeRelease(conn, lockID), nil
}

func makeRelease(conn *postgres.Conn, lockID int64) func() {
	once := sync.Once{}
	return func() {
		once.Do(func() {
			unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer unlockCancel()
			_, err := conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", lockID)
			if err != nil {
				log.Errorf("Failed to release advisory lock %d: %v", lockID, err)
			} else {
				log.Infof("Advisory lock %d released.", lockID)
			}
			conn.Release()
		})
	}
}
