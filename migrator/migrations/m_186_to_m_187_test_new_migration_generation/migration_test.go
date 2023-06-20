//go:build sql_integration

package m186tom187

import (
    "context"
    "testing"

    pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
    ctx = sac.WithAllAccess(context.Background())
)

type migrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres
}

func TestMigration(t *testing.T) {
    suite.Run(T, new(migrationTestSuite))
}


func (s *migrationTestSuite) SetupSuite() {
	s.db = pghelper.ForT(s.T(), true)
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
	}

	s.Require().NoError(migration.Run(dbs))

    // TODO(dont-merge): instantiate any store required for the post-migration dataset pull from DB

	// TODO(dont-merge): pull the post-migration dataset from DB

	// TODO(dont-merge): validate that the post-migration dataset has the expected content

}

// TODO(dont-merge): remove any pending TODO
