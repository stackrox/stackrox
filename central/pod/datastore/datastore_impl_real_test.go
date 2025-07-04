//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	podSearch "github.com/stackrox/rox/central/pod/datastore/internal/search"
	podStore "github.com/stackrox/rox/central/pod/datastore/internal/store/postgres"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopDataStore "github.com/stackrox/rox/central/processlisteningonport/datastore"
	plopPostgresStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
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

	postgres *pgtest.TestPostgres
	filter   filter.Filter

	ctx          context.Context
	plopAndPiCtx context.Context
}

func (s *PodDatastoreSuite) SetupSuite() {

	s.ctx = sac.WithAllAccess(context.Background())

	s.plopAndPiCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))
}

func (s *PodDatastoreSuite) SetupTest() {
	s.postgres = pgtest.ForT(s.T())

	podStorage := podStore.New(s.postgres.DB)
	podSearcher := podSearch.New(podStorage)

	var plopStorage = plopPostgresStore.NewFullStore(s.postgres.DB)

	indicatorStorage := processIndicatorStorage.New(s.postgres.DB)

	s.indicatorDataStore = processIndicatorDataStore.New(
		indicatorStorage, plopStorage, nil)

	s.plopDS = plopDataStore.New(plopStorage, s.indicatorDataStore, s.postgres.DB)

	s.filter = filter.NewFilter(5, 5, []int{5, 4, 3, 2, 1})

	s.datastore = newDatastoreImpl(podStorage, podSearcher, s.indicatorDataStore, s.plopDS, s.filter)
}

func (s *PodDatastoreSuite) getProcessIndicatorsFromDB() []*storage.ProcessIndicator {
	indicatorsFromDB := []*storage.ProcessIndicator{}
	err := s.indicatorDataStore.WalkAll(s.plopAndPiCtx,
		func(processIndicator *storage.ProcessIndicator) error {
			indicatorsFromDB = append(indicatorsFromDB, processIndicator)
			return nil
		})

	s.NoError(err)

	return indicatorsFromDB
}

// Add plops, process indicators, pods. Delete one of the pods.
// Check that the correct pod, process indicators, and plops are deleted.
func (s *PodDatastoreSuite) TestRemovePod() {

	deploymentDS, err := deploymentStore.GetTestPostgresDataStore(s.T(), s.postgres.DB)
	s.Nil(err)
	s.NoError(deploymentDS.UpsertDeployment(s.ctx, &storage.Deployment{Id: fixtureconsts.Deployment1, Namespace: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1}))

	s.NoError(s.datastore.UpsertPod(s.ctx, fixtures.GetPod1()))

	indicator1 := fixtures.GetProcessIndicator1()
	indicator2 := fixtures.GetProcessIndicator2()
	indicator3 := fixtures.GetProcessIndicator3()
	indicators := []*storage.ProcessIndicator{indicator1, indicator2, indicator3}

	s.NoError(s.indicatorDataStore.AddProcessIndicators(s.plopAndPiCtx, indicators...))

	openPlopObject1 := fixtures.GetOpenPlopObject1()
	openPlopObject2 := fixtures.GetOpenPlopObject2()
	openPlopObject3 := fixtures.GetOpenPlopObject3()

	plopObjects := []*storage.ProcessListeningOnPortFromSensor{openPlopObject1, openPlopObject2, openPlopObject3}
	s.NoError(s.datastore.plops.AddProcessListeningOnPort(
		s.plopAndPiCtx, fixtureconsts.Cluster1, plopObjects...))

	// Fetch inserted PLOP
	newPlops, err := s.datastore.plops.GetProcessListeningOnPort(
		s.ctx, fixtureconsts.Deployment1)
	s.NoError(err)

	s.Len(newPlops, 3)

	s.NoError(s.datastore.RemovePod(s.ctx, fixtureconsts.PodUID1))

	// Fetch inserted PLOP back after deleting pod
	newPlops, err = s.datastore.plops.GetProcessListeningOnPort(
		s.ctx, fixtureconsts.Deployment1)
	s.NoError(err)

	// Verify that the correct listening endpoints have been deleted
	s.Len(newPlops, 1)

	protoassert.Equal(s.T(), newPlops[0], &storage.ProcessListeningOnPort{
		PodId:         fixtureconsts.PodName2,
		PodUid:        fixtureconsts.PodUID2,
		DeploymentId:  fixtureconsts.Deployment1,
		ClusterId:     fixtureconsts.Cluster1,
		Namespace:     fixtureconsts.Namespace1,
		ContainerName: "containername",
		Endpoint: &storage.ProcessListeningOnPort_Endpoint{
			Port:     80,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
		Signal: &storage.ProcessSignal{
			ContainerId:  "containerid",
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "bin",
			LineageInfo: []*storage.ProcessSignal_LineageInfo{
				{
					ParentUid:          22,
					ParentExecFilePath: "/bin/bash",
				},
				{
					ParentUid:          28,
					ParentExecFilePath: "/bin/curl",
				},
			},
		},
	})

	// Verify that the correct process indicators have been deleted
	indicatorsFromDB := s.getProcessIndicatorsFromDB()
	s.Len(indicatorsFromDB, 1)

	protoassert.Equal(s.T(), indicatorsFromDB[0], indicator3)
}
