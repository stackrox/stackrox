//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	podSearch "github.com/stackrox/rox/central/pod/datastore/internal/search"
	podStore "github.com/stackrox/rox/central/pod/store/postgres"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopDataStore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store"
	plopPostgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stretchr/testify/suite"
)

func TestPodDatastore(t *testing.T) {
	suite.Run(t, new(PodDatastoreSuite))
}

type PodDatastoreSuite struct {
	suite.Suite

	datastore          *datastoreImpl
	indicatorDataStore processIndicatorDataStore.DataStore
	plopDS             plopDataStore.DataStore
	plopStorage        plopStore.Store

	postgres *pgtest.TestPostgres
	filter   filter.Filter

	hasWriteCtx         context.Context
	ctx                 context.Context
	processIndicatorCtx context.Context
}

func (s *PodDatastoreSuite) SetupSuite() {
	s.postgres = pgtest.ForT(s.T())

	podStorage := podStore.New(s.postgres.DB)
	podIndexer := processIndicatorStorage.NewIndexer(s.postgres.DB)
	podSearcher := podSearch.New(podStorage, podIndexer)

	s.plopStorage = plopPostgresStore.NewFullStore(s.postgres.DB)

	indicatorStorage := processIndicatorStorage.New(s.postgres.DB)
	indicatorIndexer := processIndicatorStorage.NewIndexer(s.postgres.DB)
	indicatorSearcher := processIndicatorSearch.New(indicatorStorage, indicatorIndexer)

	s.indicatorDataStore, _ = processIndicatorDataStore.New(
		indicatorStorage, s.plopStorage, indicatorSearcher, nil)

	//	s.plopDS = nil
	s.plopDS = plopDataStore.New(s.plopStorage, s.indicatorDataStore)

	s.filter = filter.NewFilter(5, 5, []int{5, 4, 3, 2, 1})

	s.datastore = newDatastoreImpl(podStorage, podSearcher, s.indicatorDataStore, s.plopDS, s.filter)

	// s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
	//       sac.AllowFixedScopes(
	//               sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
	//	resources.Deployment)

	testContexts := testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.Deployment)

	s.ctx = testContexts[testutils.UnrestrictedReadWriteCtx]

	s.processIndicatorCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))
}

func (s *PodDatastoreSuite) TearDownSuite() {
}

func (s *PodDatastoreSuite) SetupTest() {

}

func (s *PodDatastoreSuite) TearDownTest() {
}

func (s *PodDatastoreSuite) getProcessIndicatorsFromDB() []*storage.ProcessIndicator {
	indicatorsFromDB := []*storage.ProcessIndicator{}
	err := s.indicatorDataStore.WalkAll(s.processIndicatorCtx,
		func(processIndicator *storage.ProcessIndicator) error {
			indicatorsFromDB = append(indicatorsFromDB, processIndicator)
			return nil
		})

	s.NoError(err)

	return indicatorsFromDB
}

// Add plops, process indicators, pods. Delete one of the pods.
// Check that the correct pod, process indicators, and plops are deleted.
func (s PodDatastoreSuite) TestRemovePod() {


	s.NoError(s.datastore.UpsertPod(s.ctx, fixtures.GetPod1()))

	indicator1 := fixtures.GetProcessIndicator1()
	indicator2 := fixtures.GetProcessIndicator2()
	indicator3 := fixtures.GetProcessIndicator3()
	indicators := []*storage.ProcessIndicator{indicator1, indicator2, indicator3}

	s.NoError(s.indicatorDataStore.AddProcessIndicators(s.processIndicatorCtx, indicators...))

	openPlopObject1 := fixtures.GetOpenPlopObject1()
	openPlopObject2 := fixtures.GetOpenPlopObject2()
	openPlopObject3 := fixtures.GetOpenPlopObject3()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{openPlopObject1, openPlopObject2, openPlopObject3}
	s.NoError(s.datastore.plops.AddProcessListeningOnPort(
		s.processIndicatorCtx, plopObjects...))

	// Fetch inserted PLOP
	newPlops, err := s.datastore.plops.GetProcessListeningOnPort(
		s.processIndicatorCtx, fixtureconsts.Deployment1)
	s.NoError(err)

	s.Len(newPlops, 3)

	s.NoError(s.datastore.RemovePod(s.ctx, fixtureconsts.PodUID1))

	// Fetch inserted PLOP back after deleting pod
	newPlops, err = s.datastore.plops.GetProcessListeningOnPort(
		s.processIndicatorCtx, fixtureconsts.Deployment1)
	s.NoError(err)

	// Verify that the correct listening endpoints have been deleted
	s.Len(newPlops, 1)

	s.Equal(*newPlops[0], storage.ProcessListeningOnPort{
		ContainerName: "test_container1",
		PodId:         fixtureconsts.PodName2,
		PodUid:        fixtureconsts.PodUID2,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     fixtureconsts.Namespace1,
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     80,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		Signal: &storage.ProcessSignal{
			Name:         "test_process2",
			Args:         "test_arguments2",
			ExecFilePath: "test_path2",
		},
	})

	// Verify that the correct process indicators have been deleted
	indicatorsFromDB := s.getProcessIndicatorsFromDB()
	s.Len(indicatorsFromDB, 1)

	s.Equal(*indicatorsFromDB[0], indicator3)

}
