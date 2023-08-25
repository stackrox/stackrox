//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	activeComponent "github.com/stackrox/rox/central/activecomponent/datastore"
	alertStore "github.com/stackrox/rox/central/alert/datastore"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterHealthPostgresStore "github.com/stackrox/rox/central/cluster/store/clusterhealth/postgres"
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	podStore "github.com/stackrox/rox/central/pod/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var (
	orphanWindow = 30 * time.Minute
)

type PostgresPruningSuite struct {
	suite.Suite
	ctx    context.Context
	testDB *pgtest.TestPostgres
}

func TestPruning(t *testing.T) {
	suite.Run(t, new(PostgresPruningSuite))
}

func (s *PostgresPruningSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *PostgresPruningSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *PostgresPruningSuite) TestPruneActiveComponents() {
	depStore, _ := deploymentStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	acDS, err := activeComponent.NewForTestOnly(s.T(), s.testDB.DB)
	s.NoError(err)

	// Create and save a deployment
	deployment := &storage.Deployment{
		Id:   fixtureconsts.Deployment1,
		Name: "TestDeployment",
	}
	err = depStore.UpsertDeployment(s.ctx, deployment)
	s.Nil(err)

	activeComponents := []*storage.ActiveComponent{
		{
			Id:           "test1",
			DeploymentId: fixtureconsts.Deployment1,
		},
		{
			Id:           "test2",
			DeploymentId: fixtureconsts.Deployment2,
		},
		{
			Id:           "test3",
			DeploymentId: fixtureconsts.Deployment2,
		},
	}
	err = acDS.UpsertBatch(s.ctx, activeComponents)
	s.Nil(err)

	exists, err := acDS.Exists(s.ctx, "test1")
	s.Nil(err)
	s.True(exists)
	exists, err = acDS.Exists(s.ctx, "test2")
	s.Nil(err)
	s.True(exists)

	PruneActiveComponents(s.ctx, s.testDB.DB)

	exists, err = acDS.Exists(s.ctx, "test1")
	s.Nil(err)
	s.True(exists)
	exists, err = acDS.Exists(s.ctx, "test2")
	s.Nil(err)
	s.False(exists)
}

func (s *PostgresPruningSuite) TestPruneClusterHealthStatuses() {
	clusterDS, err := clusterStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Nil(err)

	clusterID, err := clusterDS.AddCluster(s.ctx, &storage.Cluster{Name: "testCluster", MainImage: "docker.io/stackrox/rox:latest"})
	s.Nil(err)

	clusterHealthStore := clusterHealthPostgresStore.New(s.testDB.DB)
	healthStatuses := []*storage.ClusterHealthStatus{
		{
			Id:                 clusterID,
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			Id:                    fixtureconsts.Cluster1,
			SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
			CollectorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		},
		{
			Id:                 fixtureconsts.Cluster2,
			SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		},
	}

	err = clusterHealthStore.UpsertMany(s.ctx, healthStatuses)
	s.Nil(err)

	count, err := clusterHealthStore.Count(s.ctx)
	s.Nil(err)
	s.Equal(count, 3)
	exists, err := clusterHealthStore.Exists(s.ctx, fixtureconsts.Cluster2)
	s.Nil(err)
	s.True(exists)

	PruneClusterHealthStatuses(s.ctx, s.testDB.DB)

	count, err = clusterHealthStore.Count(s.ctx)
	s.Nil(err)
	s.Equal(count, 1)
	exists, err = clusterHealthStore.Exists(s.ctx, fixtureconsts.Cluster2)
	s.Nil(err)
	s.False(exists)
}

func (s *PostgresPruningSuite) TestGetOrphanedAlertIDs() {
	alertDS, err := alertStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Nil(err)

	deploymentDS, err := deploymentStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Nil(err)

	deploymentID := "2c507da1-b882-48cc-8143-b74e14c5cd4f"
	s.NoError(deploymentDS.UpsertDeployment(s.ctx, &storage.Deployment{Id: deploymentID}))

	now := types.TimestampNow()
	old := protoconv.ConvertTimeToTimestamp(time.Now().Add(-2 * orphanWindow))

	cases := []struct {
		name           string
		alert          *storage.Alert
		shouldBePruned bool
	}{
		{
			name: "base",
			alert: &storage.Alert{
				Id:             uuid.NewV4().String(),
				LifecycleStage: storage.LifecycleStage_DEPLOY,
				State:          storage.ViolationState_ACTIVE,
				Time:           old,
				Entity: &storage.Alert_Deployment_{
					Deployment: &storage.Alert_Deployment{
						Id: "i-do-not-exist",
					},
				},
			},
			shouldBePruned: true,
		},
		{
			name: "matches deployment id",
			alert: &storage.Alert{
				Id:             uuid.NewV4().String(),
				LifecycleStage: storage.LifecycleStage_DEPLOY,
				State:          storage.ViolationState_ACTIVE,
				Time:           old,
				Entity: &storage.Alert_Deployment_{
					Deployment: &storage.Alert_Deployment{
						Id: deploymentID,
					},
				},
			},
			shouldBePruned: false,
		},
		{
			name: "not in orphan window",
			alert: &storage.Alert{
				Id:             uuid.NewV4().String(),
				LifecycleStage: storage.LifecycleStage_DEPLOY,
				State:          storage.ViolationState_ACTIVE,
				Time:           now,
				Entity: &storage.Alert_Deployment_{
					Deployment: &storage.Alert_Deployment{
						Id: "i-do-not-exist",
					},
				},
			},
			shouldBePruned: false,
		},
		{
			name: "not the right state",
			alert: &storage.Alert{
				Id:             uuid.NewV4().String(),
				LifecycleStage: storage.LifecycleStage_DEPLOY,
				State:          storage.ViolationState_RESOLVED,
				Time:           old,
				Entity: &storage.Alert_Deployment_{
					Deployment: &storage.Alert_Deployment{
						Id: "i-do-not-exist",
					},
				},
			},
			shouldBePruned: false,
		},
		{
			name: "not the right lifecycle",
			alert: &storage.Alert{
				Id:             uuid.NewV4().String(),
				LifecycleStage: storage.LifecycleStage_RUNTIME,
				State:          storage.ViolationState_RESOLVED,
				Time:           old,
				Entity: &storage.Alert_Deployment_{
					Deployment: &storage.Alert_Deployment{
						Id: "i-do-not-exist",
					},
				},
			},
			shouldBePruned: false,
		},
	}
	for _, c := range cases {
		s.Run(c.name, func() {
			s.NoError(alertDS.UpsertAlert(s.ctx, c.alert))
			idsToResolve, err := GetOrphanedAlertIDs(s.ctx, s.testDB.DB, orphanWindow)
			s.NoError(err)
			if c.shouldBePruned {
				s.Contains(idsToResolve, c.alert.Id)
			} else {
				s.NotContains(idsToResolve, c.alert.Id)
			}
			s.NoError(alertDS.DeleteAlerts(s.ctx, c.alert.GetId()))
		})
	}
}

func (s *PostgresPruningSuite) TestGetOrphanedPodIDs() {
	podDS, err := podStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Nil(err)

	clusterDS, err := clusterStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Nil(err)

	clusterID1, err := clusterDS.AddCluster(s.ctx, &storage.Cluster{Name: "testOrphanPodCluster1", MainImage: "docker.io/stackrox/rox:latest"})
	s.Nil(err)

	clusterID2, err := clusterDS.AddCluster(s.ctx, &storage.Cluster{Name: "testOrphanPodCluster2", MainImage: "docker.io/stackrox/rox:latest"})
	s.Nil(err)

	// Add some pods to Cluster 1
	cluster1PodCount := 20
	cluster2PodCount := 15

	s.addSomePods(podDS, clusterID1, cluster1PodCount)
	s.addSomePods(podDS, clusterID2, cluster2PodCount)

	// No pods orphaned
	idsToPrune, err := GetOrphanedPodIDs(s.ctx, s.testDB.DB)
	s.Nil(err)
	s.Equal(len(idsToPrune), 0)

	// cluster 2 pods orphaned
	err = clusterDS.RemoveCluster(s.ctx, clusterID2, nil)
	s.Nil(err)
	idsToPrune, err = GetOrphanedPodIDs(s.ctx, s.testDB.DB)
	s.Nil(err)
	s.Equal(len(idsToPrune), cluster2PodCount)
}

func (s *PostgresPruningSuite) addSomePods(podDS podStore.DataStore, clusterID string, numberPods int) {
	for i := 0; i < numberPods; i++ {
		pod := &storage.Pod{
			Id:        uuid.NewV4().String(),
			ClusterId: clusterID,
		}
		err := podDS.UpsertPod(s.ctx, pod)
		s.Nil(err)
	}
}

func newIndicatorWithDeployment(id string, age time.Duration, deploymentID string) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:            id,
		DeploymentId:  deploymentID,
		ContainerName: "",
		PodId:         "",
		Signal: &storage.ProcessSignal{
			Time: timestampNowMinus(age),
		},
	}
}

func newIndicatorWithDeploymentAndPod(id string, age time.Duration, deploymentID, podUID string) *storage.ProcessIndicator {
	indicator := newIndicatorWithDeployment(id, age, deploymentID)
	indicator.PodUid = podUID
	return indicator
}

func timestampNowMinus(t time.Duration) *types.Timestamp {
	return protoconv.ConvertTimeToTimestamp(time.Now().Add(-t))
}

func (s *PostgresPruningSuite) TestRemoveOrphanedProcesses() {
	cases := []struct {
		name              string
		initialProcesses  []*storage.ProcessIndicator
		deployments       set.FrozenStringSet
		pods              set.FrozenStringSet
		expectedDeletions []string
	}{
		{
			name: "no deployments nor pods - remove all old indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment5, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(),
			pods:              set.NewFrozenStringSet(),
			expectedDeletions: []string{fixtureconsts.ProcessIndicatorID1, fixtureconsts.ProcessIndicatorID2, fixtureconsts.ProcessIndicatorID3},
		},
		{
			name: "no deployments nor pods - remove no new orphaned indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 20*time.Minute, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment5, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 20*time.Minute, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(),
			pods:              set.NewFrozenStringSet(),
			expectedDeletions: nil,
		},
		{
			name: "all pods separate deployments - remove no indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment5, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment6, fixtureconsts.Deployment5, fixtureconsts.Deployment3),
			pods:              set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3),
			expectedDeletions: nil,
		},
		{
			name: "all pods same deployment - remove no indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment6),
			pods:              set.NewFrozenStringSet(fixtureconsts.PodUID1, fixtureconsts.PodUID2, fixtureconsts.PodUID3),
			expectedDeletions: nil,
		},
		{
			name: "some pods separate deployments - remove some indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment5, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment3, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment3),
			pods:              set.NewFrozenStringSet(fixtureconsts.PodUID3),
			expectedDeletions: []string{fixtureconsts.ProcessIndicatorID1},
		},
		{
			name: "some pods same deployment - remove some indicators",
			initialProcesses: []*storage.ProcessIndicator{
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID1, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID1),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID2, 20*time.Minute, fixtureconsts.Deployment6, fixtureconsts.PodUID2),
				newIndicatorWithDeploymentAndPod(fixtureconsts.ProcessIndicatorID3, 1*time.Hour, fixtureconsts.Deployment6, fixtureconsts.PodUID3),
			},
			deployments:       set.NewFrozenStringSet(fixtureconsts.Deployment6),
			pods:              set.NewFrozenStringSet(fixtureconsts.PodUID3),
			expectedDeletions: []string{fixtureconsts.ProcessIndicatorID1},
		},
	}
	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			// Add deployments if necessary
			deploymentDS, err := deploymentStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
			s.Nil(err)
			for _, deploymentID := range c.deployments.AsSlice() {
				s.NoError(deploymentDS.UpsertDeployment(s.ctx, &storage.Deployment{Id: deploymentID, ClusterId: fixtureconsts.Cluster1}))
			}

			podDS, err := podStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
			s.Nil(err)
			for _, podID := range c.pods.AsSlice() {
				err := podDS.UpsertPod(s.ctx, &storage.Pod{Id: podID, ClusterId: fixtureconsts.Cluster1})
				s.Nil(err)
			}

			processDatastore, err := processIndicatorDatastore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
			s.Nil(err)
			s.NoError(processDatastore.AddProcessIndicators(s.ctx, c.initialProcesses...))
			countFromDB, err := processDatastore.Count(s.ctx, nil)
			s.NoError(err)
			s.Equal(len(c.initialProcesses), countFromDB)

			PruneOrphanedProcessIndicators(s.ctx, s.testDB.DB, orphanWindow)
			countFromDB, err = processDatastore.Count(s.ctx, nil)
			s.NoError(err)
			s.Equal(len(c.initialProcesses)-len(c.expectedDeletions), countFromDB)

			// Cleanup
			var cleanupIDs []string
			for _, process := range c.initialProcesses {
				cleanupIDs = append(cleanupIDs, process.Id)
			}
			s.NoError(processDatastore.RemoveProcessIndicators(s.ctx, cleanupIDs))

			for _, deploymentID := range c.deployments.AsSlice() {
				s.NoError(deploymentDS.RemoveDeployment(s.ctx, fixtureconsts.Cluster1, deploymentID))
			}

			for _, podID := range c.pods.AsSlice() {
				s.NoError(podDS.RemovePod(s.ctx, podID))
			}
		})
	}
}
