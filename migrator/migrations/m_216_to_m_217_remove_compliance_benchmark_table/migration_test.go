//go:build sql_integration

package m216tom217

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_216_to_m_217_remove_compliance_benchmark_table/schema/old"
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

	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableComplianceOperatorBenchmarkV2Stmt)

	var num int64
	var err error

	// Ensure tables exist.
	err = dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", oldSchema.ComplianceOperatorBenchmarkV2TableName).Scan(&num)
	s.Require().NoError(err)

	err = dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", oldSchema.ComplianceOperatorBenchmarkV2ProfilesTableName).Scan(&num)
	s.Require().NoError(err)

	s.Require().NoError(migration.Run(dbs))

	// Ensure tables are deleted.
	err = dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", oldSchema.ComplianceOperatorBenchmarkV2TableName).Scan(&num)
	s.Require().ErrorIs(err, pgx.ErrNoRows)

	err = dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", oldSchema.ComplianceOperatorBenchmarkV2ProfilesTableName).Scan(&num)
	s.Require().ErrorIs(err, pgx.ErrNoRows)
}
