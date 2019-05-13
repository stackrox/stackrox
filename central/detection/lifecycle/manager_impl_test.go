package lifecycle

import (
	"math/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	processWhitelistDataStoreMocks "github.com/stackrox/rox/central/processwhitelist/datastore/mocks"
	reprocessorMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite

	whitelists  *processWhitelistDataStoreMocks.MockDataStore
	reprocessor *reprocessorMocks.MockLoop
	manager     *managerImpl

	mockCtrl *gomock.Controller
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.whitelists = processWhitelistDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.reprocessor = reprocessorMocks.NewMockLoop(suite.mockCtrl)
	suite.manager = &managerImpl{whitelists: suite.whitelists, reprocessor: suite.reprocessor}
}

func (suite *ManagerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func makeIndicator() (*storage.ProcessWhitelistKey, *storage.ProcessIndicator) {
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
		Lineage:      []string{uuid.NewV4().String()},
	}

	indicator := &storage.ProcessIndicator{
		Id:            uuid.NewV4().String(),
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: uuid.NewV4().String(),
		PodId:         uuid.NewV4().String(),
		EmitTimestamp: types.TimestampNow(),
		Signal:        signal,
	}
	key := &storage.ProcessWhitelistKey{
		DeploymentId:  indicator.DeploymentId,
		ContainerName: indicator.ContainerName,
	}
	return key, indicator
}

func (suite *ManagerTestSuite) TestWhitelistNotFound() {
	envIsolator := testutils.NewEnvIsolator(suite.T())
	defer envIsolator.RestoreAll()

	envIsolator.Setenv(env.WhitelistGenerationDuration.EnvVar(), time.Millisecond.String())
	key, indicator := makeIndicator()
	elements := fixtures.MakeWhitelistItems(indicator.GetSignal().GetExecFilePath())
	suite.whitelists.EXPECT().GetProcessWhitelist(gomock.Any(), key).Return(nil, nil)
	suite.whitelists.EXPECT().UpsertProcessWhitelist(gomock.Any(), key, elements, true).Return(nil, nil)
	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(indicator.GetDeploymentId()).Times(2)
	_, _, err := suite.manager.checkWhitelist(indicator)
	suite.NoError(err)
	time.Sleep(whitelistLockingGracePeriod + 2*time.Second)
	suite.mockCtrl.Finish()

	suite.mockCtrl = gomock.NewController(suite.T())
	expectedError := errors.Errorf("Expected error")
	suite.whitelists.EXPECT().GetProcessWhitelist(gomock.Any(), key).Return(nil, expectedError)
	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(indicator.GetDeploymentId())
	_, _, err = suite.manager.checkWhitelist(indicator)
	suite.Equal(expectedError, err)
	suite.mockCtrl.Finish()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.whitelists.EXPECT().GetProcessWhitelist(gomock.Any(), key).Return(nil, nil)
	suite.whitelists.EXPECT().UpsertProcessWhitelist(gomock.Any(), key, elements, true).Return(nil, expectedError)
	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(indicator.GetDeploymentId())
	_, _, err = suite.manager.checkWhitelist(indicator)
	suite.Equal(expectedError, err)
}

func (suite *ManagerTestSuite) TestWhitelistShouldBeUpdated() {
	key, indicator := makeIndicator()
	whitelist := &storage.ProcessWhitelist{}
	elements := fixtures.MakeWhitelistItems(indicator.Signal.GetExecFilePath())
	suite.whitelists.EXPECT().GetProcessWhitelist(gomock.Any(), key).Return(whitelist, nil)
	suite.whitelists.EXPECT().UpdateProcessWhitelistElements(gomock.Any(), key, elements, nil, true).Return(nil, nil)
	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(indicator.GetDeploymentId())
	_, _, err := suite.manager.checkWhitelist(indicator)
	suite.NoError(err)

	expectedError := errors.Errorf("Expected error")
	suite.whitelists.EXPECT().GetProcessWhitelist(gomock.Any(), key).Return(whitelist, nil)
	suite.whitelists.EXPECT().UpdateProcessWhitelistElements(gomock.Any(), key, elements, nil, true).Return(nil, expectedError)
	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(indicator.GetDeploymentId())
	_, _, err = suite.manager.checkWhitelist(indicator)
	suite.Equal(expectedError, err)
}

func (suite *ManagerTestSuite) TestWhitelistShouldPass() {
	key, indicator := makeIndicator()
	whitelist := &storage.ProcessWhitelist{Elements: fixtures.MakeWhitelistElements(indicator.Signal.GetExecFilePath())}
	suite.whitelists.EXPECT().GetProcessWhitelist(gomock.Any(), key).Return(whitelist, nil)
	suite.reprocessor.EXPECT().ReprocessRiskForDeployments(indicator.GetDeploymentId())
	_, _, err := suite.manager.checkWhitelist(indicator)
	suite.NoError(err)
}
