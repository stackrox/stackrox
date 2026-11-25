//go:build sql_integration

package m228tom229

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
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

	_, err := s.db.DB.Exec(s.ctx, "CREATE TABLE image_cves_v2 (id varchar PRIMARY KEY, severity integer)")
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(s.ctx, "CREATE INDEX imagecvesv2_severity ON image_cves_v2 USING btree (severity)")
	s.Require().NoError(err)
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
	s.Require().True(s.indexExists(indexName), "index %s should exist before migration", indexName)

	s.Require().NoError(migration.Run(s.dbs()))
	s.Require().False(s.indexExists(indexName), "index %s should be dropped after migration", indexName)

	// Verify idempotency: DROP INDEX IF EXISTS should not error.
	s.Require().NoError(migration.Run(s.dbs()))
}
