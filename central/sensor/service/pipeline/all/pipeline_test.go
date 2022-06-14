package all

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline"
	"github.com/stackrox/stackrox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	depMock *mocks.MockFragment
	tested  pipeline.ClusterPipeline
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.depMock = mocks.NewMockFragment(suite.mockCtrl)

	suite.tested = NewClusterPipeline("clusterID", suite.depMock)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestCallsMatchingPipeline() {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{},
	}
	ctx := context.Background()

	suite.depMock.EXPECT().Match(msg).Return(true)
	suite.depMock.EXPECT().Run(ctx, "clusterID", msg, nil).Return(errors.New("some error"))

	err := suite.tested.Run(ctx, msg, nil)
	suite.Error(err, "expected the error")
}

func (suite *PipelineTestSuite) TestHandlesNoMatchingPipeline() {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{},
	}
	ctx := context.Background()

	suite.depMock.EXPECT().Match(msg).Return(false)

	err := suite.tested.Run(ctx, msg, nil)
	suite.Error(err, "expected the error")
}
