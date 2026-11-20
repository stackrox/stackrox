//go:build sql_integration

package m228tom229

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	oldSchema "github.com/stackrox/rox/migrator/migrations/m_228_to_m_229_drop_compliance_v1_tables/schema/old"
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

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) tableExists(table string) bool {
	var num int64
	err := s.db.DB.QueryRow(s.ctx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", table).Scan(&num)
	return err == nil
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableComplianceConfigsStmt)
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableComplianceDomainsStmt)
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableComplianceStringsStmt)
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableComplianceRunResultsStmt)
	pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, oldSchema.CreateTableComplianceRunMetadataStmt)

	for _, table := range complianceV1Tables {
		s.True(s.tableExists(table), "table %q should exist before migration", table)
	}

	s.Require().NoError(migration.Run(dbs))

	for _, table := range complianceV1Tables {
		var num int64
		err := dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", table).Scan(&num)
		s.Require().ErrorIs(err, pgx.ErrNoRows, "table %q should be gone after migration", table)
	}

	// Idempotency: running again (e.g. roll-forward after a rollback) must be a no-op.
	s.Require().NoError(migration.Run(dbs))
	for _, table := range complianceV1Tables {
		var num int64
		err := dbs.PostgresDB.QueryRow(dbs.DBCtx, "SELECT 1 FROM information_schema.tables WHERE table_name = $1", table).Scan(&num)
		s.Require().ErrorIs(err, pgx.ErrNoRows, "table %q should still be gone after second run", table)
	}
}

func (s *migrationTestSuite) TestMigration_FreshInstall() {
	// A fresh installation won't have these tables — verify no error when tables are absent.
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}
	s.Require().NoError(migration.Run(dbs))
}
