package processindicators

import (
	"math/rand"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processwhitelist/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	whitelists *mocks.MockDataStore
	pipeline   pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.whitelists = mocks.NewMockDataStore(suite.mockCtrl)

	suite.pipeline = pipelineImpl{whitelists: suite.whitelists}
}

func (suite *PipelineTestSuite) TearDownTest() {
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

func (suite *PipelineTestSuite) TestWhitelistNotFound() {
	key, indicator := makeIndicator()
	elements := fixtures.MakeElements([]string{indicator.Signal.GetExecFilePath()})
	suite.whitelists.EXPECT().GetProcessWhitelist(key).Return(nil, nil)
	suite.whitelists.EXPECT().UpsertProcessWhitelist(key, elements, true).Return(nil, nil)
	err := suite.pipeline.CheckWhitelist(indicator)
	suite.NoError(err)

	expectedError := errors.Errorf("Expected error")
	suite.whitelists.EXPECT().GetProcessWhitelist(key).Return(nil, expectedError)
	err = suite.pipeline.CheckWhitelist(indicator)
	suite.Equal(expectedError, err)

	suite.whitelists.EXPECT().GetProcessWhitelist(key).Return(nil, nil)
	suite.whitelists.EXPECT().UpsertProcessWhitelist(key, elements, true).Return(nil, expectedError)
	err = suite.pipeline.CheckWhitelist(indicator)
	suite.Equal(expectedError, err)
}

func (suite *PipelineTestSuite) TestWhitelistShouldBeUpdated() {
	key, indicator := makeIndicator()
	whitelist := &storage.ProcessWhitelist{}
	elements := fixtures.MakeElements([]string{indicator.Signal.GetExecFilePath()})
	suite.whitelists.EXPECT().GetProcessWhitelist(key).Return(whitelist, nil)
	suite.whitelists.EXPECT().UpdateProcessWhitelistElements(key, elements, nil, true).Return(nil, nil)
	err := suite.pipeline.CheckWhitelist(indicator)
	suite.NoError(err)

	expectedError := errors.Errorf("Expected error")
	suite.whitelists.EXPECT().GetProcessWhitelist(key).Return(whitelist, nil)
	suite.whitelists.EXPECT().UpdateProcessWhitelistElements(key, elements, nil, true).Return(nil, expectedError)
	err = suite.pipeline.CheckWhitelist(indicator)
	suite.Equal(expectedError, err)
}

func (suite *PipelineTestSuite) TestWhitelistShouldPass() {
	key, indicator := makeIndicator()
	element := &storage.WhitelistElement{
		Element: &storage.WhitelistItem{
			Item: &storage.WhitelistItem_ProcessName{ProcessName: indicator.Signal.GetExecFilePath()},
		},
		Auto: true,
	}
	whitelist := &storage.ProcessWhitelist{Elements: []*storage.WhitelistElement{element}}
	suite.whitelists.EXPECT().GetProcessWhitelist(key).Return(whitelist, nil)
	err := suite.pipeline.CheckWhitelist(indicator)
	suite.NoError(err)
}
