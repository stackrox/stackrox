package all

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	depMock *mocks.MockFragment
	tested  pipeline.Pipeline
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.depMock = mocks.NewMockFragment(suite.mockCtrl)

	suite.tested = NewPipeline(suite.depMock)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestCallsMatchingPipeline() {
	expectedError := fmt.Errorf("this is expected")
	msg := &central.MsgFromSensor{}

	suite.depMock.EXPECT().Match(msg).Return(true)
	suite.depMock.EXPECT().Run(msg, nil).Return(expectedError)

	err := suite.tested.Run(msg, nil)
	suite.Equal(expectedError, err, "expected the error")
}

func (suite *PipelineTestSuite) TestHandlesNoMatchingPipeline() {
	msg := &central.MsgFromSensor{}

	suite.depMock.EXPECT().Match(msg).Return(false)

	err := suite.tested.Run(msg, nil)
	suite.Error(err, "expected the error")
}
