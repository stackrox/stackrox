//go:build sql_integration

package m196tom197

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	podSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/pods"
	listeningEndpointsSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/listening_endpoints"
	processIndicatorSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/process_indicators"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	podDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/pod"
	plopDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/processlisteningonport"
	processIndicatorDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/processindicator"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/fixtures"
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

	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), podSchema.CreateTablePodsStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), listeningEndpointsSchema.CreateTableListeningEndpointsStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), processIndicatorSchema.CreateTableProcessIndicatorsStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMap() {
	podStore := podDatastore.New(s.db)
	pod := fixtures.GetPod1()
	podStore.Upsert(s.ctx, pod)
	podUIDMap, err := getPodUIDMap(s.ctx, podStore)
	s.NoError(err)

	key := getPodKey(pod.Name, pod.DeploymentId)

	s.Equal(podUIDMap[key], pod.Id)
}

func (s *migrationTestSuite) TestSetUIDsUsingPods() {
	podStore := podDatastore.New(s.db)
	plopStore := plopDatastore.New(s.db)

	pods := []*storage.Pod{
		fixtures.GetPod1(),
		fixtures.GetPod2(),
	}

	podStore.UpsertMany(s.ctx, pods)

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

	count, err := plopStore.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(6, count)

	batchSize := 4
	s.Require().NoError(setPodUIDsUsingPods(s.ctx, podStore, plopStore, batchSize))

	expectedPlops := plops
	expectedPlops[1].PodUid = pods[1].Id

	for _, expectedPlop := range expectedPlops {
		actualPlop, exists, err := plopStore.Get(s.ctx, expectedPlop.Id)
		s.Require().NoError(err)
		s.Equal(true, exists)
		s.Equal(actualPlop, expectedPlop)
	}

}

func (s *migrationTestSuite) TestSetUIDsUsingProcessIndicators() {
	processIndicatorStore := processIndicatorDatastore.New(s.db)
	plopStore := plopDatastore.New(s.db)

	processIndicators := []*storage.ProcessIndicator{
		fixtures.GetProcessIndicator4(),
		fixtures.GetProcessIndicator5(),
		fixtures.GetProcessIndicator6(),
	}

	processIndicatorStore.UpsertMany(s.ctx, processIndicators)

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

	count, err := plopStore.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(6, count)

	batchSize := 4
	s.Require().NoError(setPodUIDsUsingProcessIndicators(s.ctx, processIndicatorStore, plopStore, batchSize))

	expectedPlops := plops
	expectedPlops[2].PodUid = processIndicators[2].PodUid

	for _, expectedPlop := range expectedPlops {
		actualPlop, exists, err := plopStore.Get(s.ctx, expectedPlop.Id)
		s.Require().NoError(err)
		s.Equal(true, exists)
		s.Equal(actualPlop, expectedPlop)
	}

}

func (s *migrationTestSuite) TestMigration() {
	podStore := podDatastore.New(s.db)
	processIndicatorStore := processIndicatorDatastore.New(s.db)
	plopStore := plopDatastore.New(s.db)

	pods := []*storage.Pod{
		fixtures.GetPod1(),
		fixtures.GetPod2(),
	}

	podStore.UpsertMany(s.ctx, pods)

	processIndicators := []*storage.ProcessIndicator{
		fixtures.GetProcessIndicator4(),
		fixtures.GetProcessIndicator5(),
		fixtures.GetProcessIndicator6(),
	}

	processIndicatorStore.UpsertMany(s.ctx, processIndicators)

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

	count, err := plopStore.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(6, count)

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	expectedPlops := plops
	expectedPlops[1].PodUid = pods[1].Id
	expectedPlops[2].PodUid = processIndicators[2].PodUid

	for _, expectedPlop := range expectedPlops {
		actualPlop, exists, err := plopStore.Get(s.ctx, expectedPlop.Id)
		s.Require().NoError(err)
		s.Equal(true, exists)
		s.Equal(actualPlop, expectedPlop)
	}

}
