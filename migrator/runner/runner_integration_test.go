//go:build sql_integration

package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/types"
	migVersion "github.com/stackrox/rox/migrator/version"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/version"
	versiontest "github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/require"
)

func TestInterruptedMigrationResumes(t *testing.T) {
	old := version.GetMainVersion()
	t.Cleanup(func() { versiontest.SetMainVersion(t, old) })
	versiontest.SetMainVersion(t, "5.1.0")
	ctx := context.Background()
	source := pgtest.GetConnectionString(t)
	db := pgtest.OpenGormDB(t, source)
	t.Cleanup(func() { pgtest.CloseGormDB(t, db) })
	config, err := postgres.ParseConfig(source)
	require.NoError(t, err)
	pool, err := postgres.New(ctx, config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	databases := &types.Databases{GormDB: db, PostgresDB: pool}
	migVersion.SetVersion(ctx, db, &storage.Version{Version: "4.10.0", SeqNum: 220, MinSeqNum: 209}, true)
	ver, err := migVersion.ReadVersionGormDB(ctx, db)
	require.NoError(t, err)
	interrupt := true
	lookup := func(seq int) (types.Migration, bool) {
		return types.Migration{StartingSeqNum: seq, VersionAfter: &storage.Version{SeqNum: int32(seq + 1)}, Run: func(*types.Databases) error {
			if interrupt && seq == 222 {
				return errors.New("interrupted")
			}
			return nil
		}}, true
	}
	require.ErrorContains(t, runMigrations(databases, *ver, lookup), "interrupted")
	ver, err = migVersion.ReadVersionGormDB(ctx, db)
	require.NoError(t, err)
	require.Equal(t, 222, ver.SeqNum)
	require.Equal(t, "4.10.0", ver.MainVersion)
	require.Equal(t, 220, ver.MinimumSeqNum)
	require.NoError(t, migVersion.CheckCompatibility(ver))
	interrupt = false
	require.NoError(t, runMigrations(databases, *ver, lookup))
	ver, err = migVersion.ReadVersionGormDB(ctx, db)
	require.NoError(t, err)
	require.Equal(t, migrations.CurrentDBVersionSeqNum(), ver.SeqNum)
	require.Equal(t, "4.10.0", ver.MainVersion)
	require.NoError(t, UpdateToCurrentVersion(databases))
	ver, err = migVersion.ReadVersionGormDB(ctx, db)
	require.NoError(t, err)
	require.Equal(t, "5.1.0", ver.MainVersion)
}
