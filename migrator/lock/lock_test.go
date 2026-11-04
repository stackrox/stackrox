//go:build sql_integration

package lock

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/require"
)

func TestAcquisitionContextDoesNotReleaseLock(t *testing.T) {
	ctx := t.Context()
	config, err := postgres.ParseConfig(pgtest.GetConnectionString(t))
	require.NoError(t, err)
	pool, err := postgres.New(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	lockCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	release, err := AcquireMigrationLock(lockCtx, pool)
	require.NoError(t, err)
	defer release()
	cancel()

	// Expiry of an acquisition deadline must not end a running migration's
	// ownership of its lock. Another connection must still be excluded.
	acquired, otherRelease, err := TryAcquireMigrationLock(ctx, pool)
	require.NoError(t, err)
	if otherRelease != nil {
		defer otherRelease()
	}
	require.False(t, acquired)
	release()

	acquired, otherRelease, err = TryAcquireMigrationLock(ctx, pool)
	require.NoError(t, err)
	require.True(t, acquired)
	otherRelease()
}
