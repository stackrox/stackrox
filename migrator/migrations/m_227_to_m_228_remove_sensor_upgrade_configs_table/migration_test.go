//go:build sql_integration

package m227tom228

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_227_to_m_228_remove_sensor_upgrade_configs_table/schema/old"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableSensorUpgradeConfigsStmt)

	var num int64

	// Ensure the table exists before the migration runs.
	err := dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", oldSchema.SensorUpgradeConfigsTableName).Scan(&num)
	s.Require().NoError(err)

	s.Require().NoError(migration.Run(dbs))

	// Ensure the table is gone after the migration runs.
	err = dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", oldSchema.SensorUpgradeConfigsTableName).Scan(&num)
	s.Require().ErrorIs(err, pgx.ErrNoRows)

	// Idempotency: running again (e.g. roll-forward after a rollback) must be a no-op,
	// even when the table is already gone.
	s.Require().NoError(migration.Run(dbs))
	err = dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", oldSchema.SensorUpgradeConfigsTableName).Scan(&num)
	s.Require().ErrorIs(err, pgx.ErrNoRows)
}
