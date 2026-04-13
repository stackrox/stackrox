//go:build sql_integration

package m222tom223

import (
	"context"
	"testing"

	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
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

	// Create the deployments table with the new tombstone columns.
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), pkgSchema.CreateTableDeploymentsStmt)
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	// Verify the index does not exist before migration.
	exists, err := indexhelper.IndexExists(s.ctx, s.db, tableName, indexName, "btree")
	s.Require().NoError(err)
	s.Require().False(exists, "Index should not exist before migration")

	// Run the migration.
	s.Require().NoError(migration.Run(dbs))

	// Verify the index was created.
	s.verifyIndex()

	// Run the migration again to ensure idempotency.
	s.Assert().NoError(migration.Run(dbs))

	// Verify the index still exists.
	s.verifyIndex()
}

func (s *migrationTestSuite) verifyIndex() {
	exists, err := indexhelper.IndexExists(s.ctx, s.db, tableName, indexName, "btree")
	s.Assert().NoError(err)
	s.Assert().True(exists, "Index %s should exist on %s.%s", indexName, tableName, indexColumn)
}
