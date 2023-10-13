//go:build sql_integration

package datastore

import (
	"context"
	"testing"
	"time"

	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
        processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
        processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
        podStore "github.com/stackrox/rox/central/pod/store/postgres"
        podSearch "github.com/stackrox/rox/central/pod/datastore/internal/search"
	plopDataStore "github.com/stackrox/rox/central/processlisteningonport/datastore"
        plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stretchr/testify/suite"
)

func TestPodDatastore(t *testing.T) {
	suite.Run(t, new(podDatastoreSuite))
}

type podDatastoreSuite struct {
	suite.Suite

	datastore *datastoreImpl
	indicatorDataStore processIndicatorDataStore.DataStore
	plopDS plopDataStore.DataStore
	plopStorage plopStore.Store

	postgres *pgtest.TestPostgres
	filter       filter.Filter

	hasWriteCtx context.Context
	ctx context.Context
}

func (s *podDatastoreSuite) SetupSuite() {
	s.postgres = pgtest.ForT(s.T())

	podStorage := podStore.New(s.postgres.DB)
        podIndexer := processIndicatorStorage.NewIndexer(s.postgres.DB)
	podSearcher := podSearch.New(podStorage, podIndexer)

        s.plopStorage = plopStore.NewFullStore(s.postgres.DB)

        indicatorStorage := processIndicatorStorage.New(s.postgres.DB)
        indicatorIndexer := processIndicatorStorage.NewIndexer(s.postgres.DB)
        indicatorSearcher := processIndicatorSearch.New(indicatorStorage, indicatorIndexer)

        s.indicatorDataStore, _ = processIndicatorDataStore.New(
                indicatorStorage, s.plopStorage, indicatorSearcher, nil)

        //s.plopDS = plopDataStore.New(s.plopStorage, s.indicatorDataStore)
	s.plopDS = nil

	s.filter = filter.NewFilter(5, 5, []int{5, 4, 3, 2, 1})

	s.datastore = newDatastoreImpl(podStorage, podSearcher, s.indicatorDataStore, s.plopDS, s.filter)
	podsSAC = sac.ForResource(resources.Deployment)

	//s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
        //        sac.AllowFixedScopes(
        //                sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
	//	resources.Deployment)

	testContexts := testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.Deployment)

	s.ctx = testContexts[testutils.UnrestrictedReadWriteCtx]
}

func (s *podDatastoreSuite) TearDownSuite() {
}

func (s *podDatastoreSuite) SetupTest() {

}

func (s *podDatastoreSuite) TearDownTest() {
}

var (
        openPlopObject = storage.ProcessListeningOnPortFromSensor{
                Port:           1234,
                Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
                CloseTimestamp: nil,
                Process: &storage.ProcessIndicatorUniqueKey{
                        PodId:               fixtureconsts.PodUID1,
                        ContainerName:       "test_container1",
                        ProcessName:         "test_process1",
                        ProcessArgs:         "test_arguments1",
                        ProcessExecFilePath: "test_path1",
                },
                DeploymentId: fixtureconsts.Deployment1,
        }

        closedPlopObject = storage.ProcessListeningOnPortFromSensor{
                Port:           1234,
                Protocol:       storage.L4Protocol_L4_PROTOCOL_TCP,
                CloseTimestamp: protoconv.ConvertTimeToTimestamp(time.Now()),
                Process: &storage.ProcessIndicatorUniqueKey{
                        PodId:               fixtureconsts.PodUID1,
                        ContainerName:       "test_container1",
                        ProcessName:         "test_process1",
                        ProcessArgs:         "test_arguments1",
                        ProcessExecFilePath: "test_path1",
                },
                DeploymentId: fixtureconsts.Deployment1,
        }

)

// Add plops, process indicators, pods. Delete one of the pods.
// Check that the correct pod, process indicators, and plops are deleted.
func (s *podDatastoreSuite) TestRemovePod() {
	//plopObjects := []*storage.ProcessListeningOnPortFromSensor{&openPlopObject}
	//s.NoError(s.datastore.plops.AddProcessListeningOnPort(
	//	s.hasWriteCtx, plopObjects...))


	s.NoError(s.datastore.UpsertPod(s.ctx, fixtures.GetPod1()))
//func (ds *datastoreImpl) UpsertPod(ctx context.Context, pod *storage.Pod) error {


}
