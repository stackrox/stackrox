//go:build sql_integration

package m196tom197

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	listeningEndpointsSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/listening_endpoints"
	podSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/pods"
	processIndicatorSchema "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/schema/process_indicators"
	podDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/pod"
	processIndicatorDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/processindicator"
	plopDatastore "github.com/stackrox/rox/migrator/migrations/m_196_to_m_197_set_poduid_where_null/store/processlisteningonport"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
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
	err := podStore.Upsert(s.ctx, pod)
	s.NoError(err)
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

	err := podStore.UpsertMany(s.ctx, pods)
	s.Require().NoError(err)

	plops := []*storage.ProcessListeningOnPortStorage{
		fixtures.GetPlopStorage1(),
		fixtures.GetPlopStorage2(),
		fixtures.GetPlopStorage3(),
		fixtures.GetPlopStorage4(),
		fixtures.GetPlopStorage5(),
		fixtures.GetPlopStorage6(),
	}

	err = plopStore.UpsertMany(s.ctx, plops)
	s.Require().NoError(err)

	count, err := plopStore.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(6, count)

	batchSize := 4
	s.Require().NoError(setPodUIDsUsingPods(s.ctx, podStore, plopStore, batchSize))

	expectedPlops := plops
	expectedPlops[1].PodUid = pods[1].Id
	// plops[0] will not have its PodUid set since its deploymentid does not match either of the pods
	// plops[1] will have its PodUid set to that of pods[1] since the pod id and deployment id match
	// plops[2] will not have its PodUid set since it has not process information
	// plops[3] through plops[5] will not have their PodUids set, since they are already set.

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

	err := processIndicatorStore.UpsertMany(s.ctx, processIndicators)
	s.Require().NoError(err)

	plops := []*storage.ProcessListeningOnPortStorage{
		fixtures.GetPlopStorage1(),
		fixtures.GetPlopStorage2(),
		fixtures.GetPlopStorage3(),
		fixtures.GetPlopStorage4(),
		fixtures.GetPlopStorage5(),
		fixtures.GetPlopStorage6(),
	}

	err = plopStore.UpsertMany(s.ctx, plops)
	s.Require().NoError(err)

	count, err := plopStore.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(6, count)

	batchSize := 4
	s.Require().NoError(setPodUIDsUsingProcessIndicators(s.ctx, processIndicatorStore, plopStore, batchSize))

	expectedPlops := plops
	expectedPlops[2].PodUid = processIndicators[2].PodUid
	// plops[0] and plops[1] will not be set because they have process information and therefore
	// there should not be a matching process indicators
	// plops[2] will have its PodUid set since it has no process information, but has a matching process indicator
	// plops[3] through plops[5] will not have their PodUids set, since they are already set.

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

	err := podStore.UpsertMany(s.ctx, pods)
	s.Require().NoError(err)

	processIndicators := []*storage.ProcessIndicator{
		fixtures.GetProcessIndicator4(),
		fixtures.GetProcessIndicator5(),
		fixtures.GetProcessIndicator6(),
	}

	err = processIndicatorStore.UpsertMany(s.ctx, processIndicators)
	s.Require().NoError(err)

	plops := []*storage.ProcessListeningOnPortStorage{
		fixtures.GetPlopStorage1(),
		fixtures.GetPlopStorage2(),
		fixtures.GetPlopStorage3(),
		fixtures.GetPlopStorage4(),
		fixtures.GetPlopStorage5(),
		fixtures.GetPlopStorage6(),
	}

	err = plopStore.UpsertMany(s.ctx, plops)
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
	// This test is a combination of the above two tests so the PodUids changed in this test
	// is the changes of the individual tests above put together

	for _, expectedPlop := range expectedPlops {
		actualPlop, exists, err := plopStore.Get(s.ctx, expectedPlop.Id)
		s.Require().NoError(err)
		s.Equal(true, exists)
		s.Equal(actualPlop, expectedPlop)
	}

}
