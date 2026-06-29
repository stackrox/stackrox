//go:build sql_integration

package m225tom226

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_225_to_m_226_drop_unused_child_idx_indexes/test/schema"
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

	dbs := s.dbs()
	for _, stmt := range frozenSchema.AllCreateStmts {
		pgutils.CreateTableFromModel(dbs.DBCtx, dbs.GormDB, stmt)
	}
}

func (s *migrationTestSuite) dbs() *types.Databases {
	return &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}
}

func (s *migrationTestSuite) indexExists(name string) bool {
	var n int
	err := s.db.DB.QueryRow(s.ctx,
		"SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1", name).Scan(&n)
	if err == pgx.ErrNoRows {
		return false
	}
	s.Require().NoError(err)
	return true
}

func (s *migrationTestSuite) TestMigration() {
	dbs := s.dbs()

	// Verify every index we intend to drop actually exists.
	for _, name := range indexesToDrop {
		s.Require().True(s.indexExists(name), "index %s should exist before migration", name)
	}

	// Run the migration.
	s.Require().NoError(migration.Run(dbs))

	// Verify every index is gone.
	for _, name := range indexesToDrop {
		s.Require().False(s.indexExists(name), "index %s should be dropped after migration", name)
	}

	// Run again to verify idempotency: DROP INDEX IF EXISTS should not error.
	s.Require().NoError(migration.Run(dbs))
}
