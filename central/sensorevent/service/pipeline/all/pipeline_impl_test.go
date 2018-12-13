package all

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	depMock *mocks.MockPipeline
	proMock *mocks.MockPipeline
	netMock *mocks.MockPipeline
	namMock *mocks.MockPipeline
	secMock *mocks.MockPipeline
	cstMock *mocks.MockPipeline
	pmMock  *mocks.MockPipeline

	tested pipeline.Pipeline

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.depMock = mocks.NewMockPipeline(suite.mockCtrl)
	suite.proMock = mocks.NewMockPipeline(suite.mockCtrl)
	suite.netMock = mocks.NewMockPipeline(suite.mockCtrl)
	suite.namMock = mocks.NewMockPipeline(suite.mockCtrl)
	suite.secMock = mocks.NewMockPipeline(suite.mockCtrl)
	suite.cstMock = mocks.NewMockPipeline(suite.mockCtrl)
	suite.pmMock = mocks.NewMockPipeline(suite.mockCtrl)

	suite.tested = NewPipeline(suite.depMock,
		suite.proMock,
		suite.netMock,
		suite.namMock,
		suite.secMock,
		suite.cstMock,
		suite.pmMock)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestCallsDeploymentPipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Deployment{},
	}

	suite.depMock.EXPECT().Run(event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")
}

func (suite *PipelineTestSuite) TestCallProcessIndicationPipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_ProcessIndicator{},
	}

	suite.proMock.EXPECT().Run(event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")
}

func (suite *PipelineTestSuite) TestCallsNetworkPolicyPipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_NetworkPolicy{},
	}

	suite.netMock.EXPECT().Run(event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")
}

func (suite *PipelineTestSuite) TestCallsNamespacePipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Namespace{},
	}

	suite.namMock.EXPECT().Run(event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")
}

func (suite *PipelineTestSuite) TestCallsSecretPipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Secret{},
	}

	suite.secMock.EXPECT().Run(event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")
}

func (suite *PipelineTestSuite) TestHandlesNoType() {
	event := &v1.SensorEvent{
		Action: v1.ResourceAction_CREATE_RESOURCE,
	}

	err := suite.tested.Run(event, nil)
	suite.Error(err, "expected the error")
}
