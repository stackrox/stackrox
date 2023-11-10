//go:build sql_integration

package m198tom199

import (
	"context"
	"testing"

	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

type migrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}


func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	// TODO(dont-merge): Create the schemas and tables required for the pre-migration dataset push to DB
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}



func (s *migrationTestSuite) TestMigration() {
	// TODO(dont-merge): instantiate any store required for the pre-migration dataset push to DB

	// TODO(dont-merge): push the pre-migration dataset to DB

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	// TODO(dont-merge): instantiate any store required for the post-migration dataset pull from DB

	// TODO(dont-merge): pull the post-migration dataset from DB

	// TODO(dont-merge): validate that the post-migration dataset has the expected content

	// TODO(dont-merge): validate that pre-migration queries and statements execute against the
	// post-migration database to ensure backwards compatibility

}

// TODO(dont-merge): remove any pending TODO
