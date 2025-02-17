//go:build sql_integration

package m209tom210

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	listeningEndpointsSchema "github.com/stackrox/rox/migrator/migrations/m_209_to_m_210_remove_listening_endpoints_where_pod_uid_is_null/schema/listening_endpoints"
	plopDatastore "github.com/stackrox/rox/migrator/migrations/m_209_to_m_210_remove_listening_endpoints_where_pod_uid_is_null/store/processlisteningonport"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
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

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), listeningEndpointsSchema.CreateTableListeningEndpointsStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}



func (s *migrationTestSuite) TestMigration() {
	plopStore := plopDatastore.New(s.db)

	plops := []*storage.ProcessListeningOnPortStorage{
		fixtures.GetPlopStorage1(),
		fixtures.GetPlopStorage2(),
		fixtures.GetPlopStorage3(),
		fixtures.GetPlopStorage4(),
		fixtures.GetPlopStorage5(),
		fixtures.GetPlopStorage6(),
	}

	err := plopStore.UpsertMany(s.ctx, plops)
	s.Require().NoError(err)

	count, err := plopStore.Count(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Equal(6, count)

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	count, err = plopStore.Count(s.ctx, search.EmptyQuery())
	s.Require().NoError(err)
	s.Equal(3, count)

	expectedPlops := []*storage.ProcessListeningOnPortStorage{
		fixtures.GetPlopStorage4(),
		fixtures.GetPlopStorage5(),
		fixtures.GetPlopStorage6(),
	}

	for _, expectedPlop := range expectedPlops {
		actualPlop, exists, err := plopStore.Get(s.ctx, expectedPlop.Id)
		s.Require().NoError(err)
		s.Equal(true, exists)
		s.Equal(actualPlop.Port, expectedPlop.Port)
		s.Equal(actualPlop.Protocol, expectedPlop.Protocol)
		s.Equal(actualPlop.ProcessIndicatorId, expectedPlop.ProcessIndicatorId)
		s.Equal(actualPlop.Closed, expectedPlop.Closed)
		s.Equal(actualPlop.DeploymentId, expectedPlop.DeploymentId)
		s.Equal(actualPlop.PodUid, expectedPlop.PodUid)
	}

}
