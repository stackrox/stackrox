//go:build sql_integration

package m220tom221

import (
	"context"
	"testing"

	"github.com/stackrox/rox/migrator/migrations/m_220_to_m_221_add_traits_expires_at_column/schema"
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

	// Create the tables without the traits_expires_at column.
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableRolesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePermissionSetsStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableSimpleAccessScopesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableAuthProvidersStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableAuthMachineToMachineConfigsStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableSignatureIntegrationsStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableNotifiersStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTableGroupsStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	// Verify that the column was added to each table.
	for _, table := range tablesWithTraits {
		s.Run(table, func() {
			var exists bool
			query := `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_name = $1 AND column_name = 'traits_expires_at'
				)
			`
			row := s.db.DB.QueryRow(s.ctx, query, table)
			s.Require().NoError(row.Scan(&exists))
			s.True(exists, "traits_expires_at column should exist in table %s", table)
		})
	}
}
