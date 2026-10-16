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
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInterruptedMigrationResumes(t *testing.T) {
	for name, tc := range map[string]struct {
		missing   bool
		wantError string
	}{
		"migration fails":   {wantError: "interrupted"},
		"migration missing": {missing: true, wantError: "no migration found starting at 222"},
	} {
		t.Run(name, func(t *testing.T) {
			setMainVersion(t, "5.1.0")
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
			ver := readVersion(t, db)
			interrupt := true
			var migrationContext context.Context
			lookup := func(seq int) (types.Migration, bool) {
				if interrupt && tc.missing && seq == 222 {
					return types.Migration{}, false
				}
				return types.Migration{
					StartingSeqNum: seq,
					VersionAfter:   &storage.Version{SeqNum: int32(seq + 1)},
					Run: func(databases *types.Databases) error {
						require.NoError(t, databases.DBCtx.Err())
						if migrationContext == nil {
							migrationContext = databases.DBCtx
						} else {
							require.Same(t, migrationContext, databases.DBCtx, "reuse the authorized context across migrations")
						}
						if interrupt && seq == 222 {
							return errors.New("interrupted")
						}
						canceled, cancel := context.WithCancel(databases.DBCtx)
						cancel()
						databases.DBCtx = canceled
						return nil
					},
				}, true
			}
			require.ErrorContains(t, runMigrations(databases, *ver, lookup), tc.wantError)
			ver = readVersion(t, db)
			require.Equal(t, 222, ver.SeqNum)
			require.Equal(t, "4.10.0", ver.MainVersion)
			require.Equal(t, 220, ver.MinimumSeqNum)
			require.NoError(t, migVersion.CheckCompatibility(ver))
			interrupt = false
			migrationContext = nil
			require.NoError(t, runMigrations(databases, *ver, lookup))
			ver = readVersion(t, db)
			require.Equal(t, migrations.CurrentDBVersionSeqNum(), ver.SeqNum)
			require.Equal(t, "4.10.0", ver.MainVersion)
			require.NoError(t, UpdateToCurrentVersion(databases))
			ver = readVersion(t, db)
			require.Equal(t, "5.1.0", ver.MainVersion)
		})
	}
}

func TestRunMigrationPaths(t *testing.T) {
	setMainVersion(t, "5.1.0")
	current := migrations.CurrentDBVersionSeqNum()
	for name, tc := range map[string]struct {
		sequence  int
		wantError string
	}{
		"fresh":                     {},
		"matching sequence":         {sequence: current},
		"rollback":                  {sequence: current + 1},
		"upgrade missing migration": {sequence: 199, wantError: "no migration found starting at 199"},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			db := pgtest.OpenGormDB(t, pgtest.GetConnectionString(t))
			t.Cleanup(func() { pgtest.CloseGormDB(t, db) })
			if tc.sequence != 0 {
				migVersion.SetVersion(ctx, db, &storage.Version{Version: "5.1.0", SeqNum: int32(tc.sequence), MinSeqNum: 220}, true)
			}
			before := readVersion(t, db)
			// No SQL pool is supplied: none of these paths should write checkpoints.
			err := Run(&types.Databases{GormDB: db})
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
			} else {
				require.NoError(t, err)
			}
			after := readVersion(t, db)
			require.Equal(t, before, after)
			require.Equal(t, tc.sequence != 0, db.Migrator().HasTable("versions"))
		})
	}
}

func readVersion(t *testing.T, db *gorm.DB) *migrations.MigrationVersion {
	t.Helper()
	ver, err := migVersion.ReadVersionGormDB(context.Background(), db)
	require.NoError(t, err)
	return ver
}
