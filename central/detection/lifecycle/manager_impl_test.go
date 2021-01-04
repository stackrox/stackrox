package lifecycle

import (
	"math/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	processBaselineDataStoreMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	reprocessorMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite

	baselines   *processBaselineDataStoreMocks.MockDataStore
	reprocessor *reprocessorMocks.MockLoop
	manager     *managerImpl

	mockCtrl *gomock.Controller
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.baselines = processBaselineDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.reprocessor = reprocessorMocks.NewMockLoop(suite.mockCtrl)
	suite.manager = &managerImpl{baselines: suite.baselines, reprocessor: suite.reprocessor}
}

func (suite *ManagerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func makeIndicator() (*storage.ProcessBaselineKey, *storage.ProcessIndicator) {
	signal := &storage.ProcessSignal{
		Id:           uuid.NewV4().String(),
		ContainerId:  uuid.NewV4().String(),
		Time:         types.TimestampNow(),
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
	envIsolator := envisolator.NewEnvIsolator(suite.T())
	defer envIsolator.RestoreAll()

	envIsolator.Setenv(env.BaselineGenerationDuration.EnvVar(), time.Millisecond.String())
	key, indicator := makeIndicator()
	elements := fixtures.MakeBaselineItems(indicator.GetSignal().GetExecFilePath())
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(nil, false, nil)
	suite.baselines.EXPECT().UpsertProcessBaseline(gomock.Any(), key, elements, true).Return(nil, nil)
	_, err := suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.NoError(err)
	suite.mockCtrl.Finish()

	suite.mockCtrl = gomock.NewController(suite.T())
	expectedError := errors.New("Expected error")
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(nil, false, expectedError)
	_, err = suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.Equal(expectedError, err)
	suite.mockCtrl.Finish()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(nil, false, nil)
	suite.baselines.EXPECT().UpsertProcessBaseline(gomock.Any(), key, elements, true).Return(nil, expectedError)
	_, err = suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.Equal(expectedError, err)
}

func (suite *ManagerTestSuite) TestBaselineShouldBeUpdated() {
	key, indicator := makeIndicator()
	baseline := &storage.ProcessBaseline{}
	elements := fixtures.MakeBaselineItems(indicator.Signal.GetExecFilePath())
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(baseline, true, nil)
	suite.baselines.EXPECT().UpdateProcessBaselineElements(gomock.Any(), key, elements, nil, true).Return(nil, nil)
	_, err := suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.NoError(err)

	expectedError := errors.New("Expected error")
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(baseline, true, nil)
	suite.baselines.EXPECT().UpdateProcessBaselineElements(gomock.Any(), key, elements, nil, true).Return(nil, expectedError)
	_, err = suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
	suite.Equal(expectedError, err)
}

func (suite *ManagerTestSuite) TestBaselineShouldPass() {
	key, indicator := makeIndicator()
	baseline := &storage.ProcessBaseline{Elements: fixtures.MakeBaselineElements(indicator.Signal.GetExecFilePath())}
	suite.baselines.EXPECT().GetProcessBaseline(gomock.Any(), key).Return(baseline, true, nil)
	_, err := suite.manager.checkAndUpdateBaseline(indicatorToBaselineKey(indicator), []*storage.ProcessIndicator{indicator})
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

		manager := &managerImpl{removedPolicies: c.removedPolicies}
		manager.filterOutDisabledPolicies(&testAlerts)
		assert.Equal(t, c.expectedAlerts, testAlerts)
	}
}
