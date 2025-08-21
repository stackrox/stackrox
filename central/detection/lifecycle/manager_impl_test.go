package lifecycle

import (
	"math/rand"
	"testing"
	"time"

	"github.com/pkg/errors"
	clusterDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	queueMocks "github.com/stackrox/rox/central/deployment/queue/mocks"
	alertManagerMocks "github.com/stackrox/rox/central/detection/alertmanager/mocks"
	processBaselineDataStoreMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	reprocessorMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	clusterAutolockEnabled = &storage.Cluster{
		ManagedBy: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		HelmConfig: &storage.CompleteClusterConfig{
			DynamicConfig: &storage.DynamicClusterConfig{
				AutolockProcessBaseline: &storage.AutolockProcessBaselinesConfig{
					Enabled: true,
				},
			},
		},
	}

	clusterAutolockDisabled = &storage.Cluster{
		ManagedBy: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		HelmConfig: &storage.CompleteClusterConfig{
			DynamicConfig: &storage.DynamicClusterConfig{
				AutolockProcessBaseline: &storage.AutolockProcessBaselinesConfig{
					Enabled: false,
				},
			},
		},
	}

	clusterAutolockManualEnabled = &storage.Cluster{
		ManagedBy: storage.ManagerType_MANAGER_TYPE_MANUAL,
		DynamicConfig: &storage.DynamicClusterConfig{
			AutolockProcessBaseline: &storage.AutolockProcessBaseline{
				Enabled: true,
			},
		},
	}
)

func TestManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite

	baselines                  *processBaselineDataStoreMocks.MockDataStore
	reprocessor                *reprocessorMocks.MockLoop
	alertManager               *alertManagerMocks.MockAlertManager
	deploymentObservationQueue *queueMocks.MockDeploymentObservationQueue
	manager                    *managerImpl
	mockCtrl                   *gomock.Controller
	connectionManager          *connectionMocks.MockManager
	cluster                    *clusterDataStoreMocks.MockDataStore
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.baselines = processBaselineDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.reprocessor = reprocessorMocks.NewMockLoop(suite.mockCtrl)
	suite.alertManager = alertManagerMocks.NewMockAlertManager(suite.mockCtrl)
	suite.deploymentObservationQueue = queueMocks.NewMockDeploymentObservationQueue(suite.mockCtrl)
	suite.connectionManager = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.cluster = clusterDataStoreMocks.NewMockDataStore(suite.mockCtrl)

	suite.manager = &managerImpl{
		baselines:                  suite.baselines,
		reprocessor:                suite.reprocessor,
		alertManager:               suite.alertManager,
		deploymentObservationQueue: suite.deploymentObservationQueue,
		connectionManager:          suite.connectionManager,
		clusterDataStore:           suite.cluster,
	}
}

func (suite *ManagerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func makeIndicator() (*storage.ProcessBaselineKey, *storage.ProcessIndicator) {
	signal := &storage.ProcessSignal{
		Id:           uuid.NewV4().String(),
		ContainerId:  uuid.NewV4().String(),
		Time:         protocompat.TimestampNow(),
		Name:         uuid.NewV4().String(),
		Args:         uuid.NewV4().String(),
		ExecFilePath: uuid.NewV4().String(),
		Pid:          rand.Uint32(),
		Uid:          rand.Uint32(),
		Gid:          rand.Uint32(),
		LineageInfo: []*storage.ProcessSignal_LineageInfo{
			{
				ParentExecFilePath: uuid.NewV4().String(),
			},
		},
	}

	indicator := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: uuid.NewV4().String(),
		PodId:         uuid.NewV4().String(),
		Signal:        signal,
	}
	key := &storage.ProcessBaselineKey{
		DeploymentId:  indicator.GetDeploymentId(),
		ContainerName: indicator.GetContainerName(),
		ClusterId:     indicator.GetClusterId(),
		Namespace:     indicator.GetNamespace(),
	}
	return key, indicator
}

func (suite *ManagerTestSuite) TestBaselineNotFound() {
	suite.T().Setenv(env.BaselineGenerationDuration.EnvVar(), time.Millisecond.String())
	key, indicator := makeIndicator()
	elements := fixtures.MakeBaselineItems(indicator.GetSignal().GetExecFilePath())
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(nil, false, nil)
	suite.deploymentObservationQueue.EXPECT().InObservation(key.GetDeploymentId()).Return(false).AnyTimes()
	suite.baselines.EXPECT().UpsertProcessBaseline(gomock.Any(), key, elements, true, true).Return(nil, nil)
	_, _, err := suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.NoError(err)
	suite.mockCtrl.Finish()

	suite.mockCtrl = gomock.NewController(suite.T())
	expectedError := errors.New("Expected error")
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(nil, false, expectedError)
	_, _, err = suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.Equal(expectedError, err)
	suite.mockCtrl.Finish()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(nil, false, nil)
	suite.baselines.EXPECT().UpsertProcessBaseline(gomock.Any(), key, elements, true, true).Return(nil, expectedError)
	_, _, err = suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.Equal(expectedError, err)
}

func (suite *ManagerTestSuite) TestBaselineNotFoundInObservation() {
	suite.T().Setenv(env.BaselineGenerationDuration.EnvVar(), time.Millisecond.String())
	key, indicator := makeIndicator()
	elements := fixtures.MakeBaselineItems(indicator.GetSignal().GetExecFilePath())
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(nil, false, nil)
	suite.deploymentObservationQueue.EXPECT().InObservation(key.GetDeploymentId()).Return(true).AnyTimes()
	suite.baselines.EXPECT().UpsertProcessBaseline(gomock.Any(), key, elements, true, true).Return(nil, nil).MaxTimes(0)
	_, _, err := suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.NoError(err)
	suite.mockCtrl.Finish()
}

func (suite *ManagerTestSuite) TestBaselineShouldPass() {
	key, indicator := makeIndicator()
	baseline := &storage.ProcessBaseline{Elements: fixtures.MakeBaselineElements(indicator.Signal.GetExecFilePath())}
	suite.deploymentObservationQueue.EXPECT().InObservation(key.GetDeploymentId()).Return(false).AnyTimes()
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(baseline, true, nil)
	_, _, err := suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.NoError(err)
}

func (suite *ManagerTestSuite) TestHandleDeploymentAlerts() {
	alerts := []*storage.Alert{fixtures.GetAlert()}
	depID := alerts[0].GetDeployment().Id

	// unfortunately because the filters are in a different package and have unexported functions it cannot be tested here. Alert Manager tests should cover it
	suite.alertManager.EXPECT().
		AlertAndNotify(gomock.Any(), alerts, gomock.Any(), gomock.Any()).
		Return(set.NewStringSet(), nil)

	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(depID)

	err := suite.manager.HandleDeploymentAlerts(depID, alerts, storage.LifecycleStage_RUNTIME)
	suite.NoError(err)
}

func (suite *ManagerTestSuite) TestHandleResourceAlerts() {
	alerts := []*storage.Alert{fixtures.GetResourceAlert()}

	// unfortunately because the filters are in a different package and have unexported functions it cannot be tested here. Alert Manager tests should cover it
	suite.alertManager.EXPECT().
		AlertAndNotify(gomock.Any(), alerts, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(set.NewStringSet(), nil)

	// reprocessor.ReprocessRiskForDeployments should _not_ be called for resource alerts

	err := suite.manager.HandleResourceAlerts(alerts[0].GetResource().ClusterId, alerts, storage.LifecycleStage_RUNTIME)
	suite.NoError(err)
}

func TestFilterOutDisabledPolicies(t *testing.T) {
	alert1 := fixtures.GetAlertWithID("1")
	alert1.Policy.Id = "1"
	alert2 := fixtures.GetAlertWithID("2")
	alert2.Policy.Id = "2"
	cases := []struct {
		name            string
		initialAlerts   []*storage.Alert
		expectedAlerts  []*storage.Alert
		removedPolicies set.StringSet
	}{
		{
			initialAlerts:   nil,
			expectedAlerts:  nil,
			removedPolicies: set.NewStringSet(),
		},
		{
			initialAlerts:   nil,
			expectedAlerts:  nil,
			removedPolicies: set.NewStringSet("1", "2"),
		},
		{
			initialAlerts:   []*storage.Alert{alert1, alert2},
			expectedAlerts:  []*storage.Alert{alert1, alert2},
			removedPolicies: set.NewStringSet(),
		},
		{
			initialAlerts:   []*storage.Alert{alert1, alert2},
			expectedAlerts:  []*storage.Alert{alert1},
			removedPolicies: set.NewStringSet("2"),
		},
		{
			initialAlerts:   []*storage.Alert{alert1, alert2},
			expectedAlerts:  []*storage.Alert{},
			removedPolicies: set.NewStringSet("1", "2"),
		},
	}

	for _, c := range cases {
		var testAlerts []*storage.Alert
		testAlerts = append(testAlerts, c.initialAlerts...)

		manager := &managerImpl{removedOrDisabledPolicies: c.removedPolicies}
		manager.filterOutDisabledPolicies(&testAlerts)
		protoassert.SlicesEqual(t, c.expectedAlerts, testAlerts)
	}
}

func (suite *ManagerTestSuite) TestAutoLockProcessBaselines() {
	clusterId := fixtureconsts.Cluster1

	suite.T().Setenv(features.AutoLockProcessBaselines.EnvVar(), "true")
	suite.cluster.EXPECT().GetCluster(gomock.Any(), clusterId).Return(clusterAutolockEnabled, true, nil)
	enabled := suite.manager.isAutoLockEnabledForCluster(clusterId)
	suite.True(enabled)
}

func (suite *ManagerTestSuite) TestAutoLockProcessBaselinesFeatureFlagDisabled() {
	clusterId := fixtureconsts.Cluster1

	suite.T().Setenv(features.AutoLockProcessBaselines.EnvVar(), "false")
	enabled := suite.manager.isAutoLockEnabledForCluster(clusterId)
	suite.False(enabled)
}

func (suite *ManagerTestSuite) TestAutoLockProcessBaselinesDisabled() {
	clusterId := fixtureconsts.Cluster1

	suite.T().Setenv(features.AutoLockProcessBaselines.EnvVar(), "true")
	suite.cluster.EXPECT().GetCluster(gomock.Any(), clusterId).Return(clusterAutolockDisabled, true, nil)
	enabled := suite.manager.isAutoLockEnabledForCluster(clusterId)
	suite.False(enabled)
}

func (suite *ManagerTestSuite) TestAutoLockProcessBaselinesNoCluster() {
	clusterId := fixtureconsts.Cluster1

	suite.T().Setenv(features.AutoLockProcessBaselines.EnvVar(), "true")
	suite.cluster.EXPECT().GetCluster(gomock.Any(), clusterId).Return(nil, false, nil)
	enabled := suite.manager.isAutoLockEnabledForCluster(clusterId)
	suite.False(enabled)
}
