package all

import (
	"fmt"
	"testing"

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

	depMock *mocks.Pipeline
	proMock *mocks.Pipeline
	netMock *mocks.Pipeline
	namMock *mocks.Pipeline
	secMock *mocks.Pipeline
	cstMock *mocks.Pipeline

	tested pipeline.Pipeline
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.depMock = &mocks.Pipeline{}
	suite.proMock = &mocks.Pipeline{}
	suite.netMock = &mocks.Pipeline{}
	suite.namMock = &mocks.Pipeline{}
	suite.secMock = &mocks.Pipeline{}
	suite.cstMock = &mocks.Pipeline{}

	suite.tested = NewPipeline(suite.depMock,
		suite.proMock,
		suite.netMock,
		suite.namMock,
		suite.secMock,
		suite.cstMock)
}

func (suite *PipelineTestSuite) TestCallsDeploymentPipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Deployment{},
	}

	suite.depMock.On("Run", event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")

	suite.assertExpectationsMet()
}

func (suite *PipelineTestSuite) TestCallProcessIndicationPipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_ProcessIndicator{},
	}

	suite.proMock.On("Run", event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")

	suite.assertExpectationsMet()
}

func (suite *PipelineTestSuite) TestCallsNetworkPolicyPipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_NetworkPolicy{},
	}

	suite.netMock.On("Run", event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")

	suite.assertExpectationsMet()
}

func (suite *PipelineTestSuite) TestCallsNamespacePipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Namespace{},
	}

	suite.namMock.On("Run", event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")

	suite.assertExpectationsMet()
}

func (suite *PipelineTestSuite) TestCallsSecretPipeline() {
	expectedError := fmt.Errorf("this is expected")
	event := &v1.SensorEvent{
		Action:   v1.ResourceAction_CREATE_RESOURCE,
		Resource: &v1.SensorEvent_Secret{},
	}

	suite.secMock.On("Run", event, nil).Return(expectedError)

	err := suite.tested.Run(event, nil)
	suite.Equal(expectedError, err, "expected the error")

	suite.assertExpectationsMet()
}

func (suite *PipelineTestSuite) TestHandlesNoType() {
	event := &v1.SensorEvent{
		Action: v1.ResourceAction_CREATE_RESOURCE,
	}

	err := suite.tested.Run(event, nil)
	suite.Error(err, "expected the error")

	suite.assertExpectationsMet()
}

func (suite *PipelineTestSuite) assertExpectationsMet() {
	suite.depMock.AssertExpectations(suite.T())
	suite.proMock.AssertExpectations(suite.T())
	suite.netMock.AssertExpectations(suite.T())
	suite.namMock.AssertExpectations(suite.T())
	suite.secMock.AssertExpectations(suite.T())
}
