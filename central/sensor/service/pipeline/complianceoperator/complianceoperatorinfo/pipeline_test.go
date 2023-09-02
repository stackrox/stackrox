package complianceoperatorinfo

import (
	"context"
	"testing"

	managerMocks "github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager/mocks"
	coIntegrationMocks "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite
	pipeline        *pipelineImpl
	manager         *managerMocks.MockManager
	coIntegrationDS *coIntegrationMocks.MockDataStore
	mockCtrl        *gomock.Controller
}

func (suite *PipelineTestSuite) SetupSuite() {
	suite.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		suite.T().Skip("Skip tests when ComplianceEnhancements disabled")
		suite.T().SkipNow()
	}
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.manager = managerMocks.NewMockManager(suite.mockCtrl)
	suite.coIntegrationDS = coIntegrationMocks.NewMockDataStore(suite.mockCtrl)
	suite.pipeline = NewPipeline(suite.manager).(*pipelineImpl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestComplianceInfoMsgFromSensor() {
	complianceMsg := &central.ComplianceOperatorInfo{
		Version:   "22",
		Namespace: fixtureconsts.Namespace1,
		TotalDesiredPodsOpt: &central.ComplianceOperatorInfo_TotalDesiredPods{
			TotalDesiredPods: 5,
		},
		TotalReadyPodsOpt: &central.ComplianceOperatorInfo_TotalReadyPods{
			TotalReadyPods: 2,
		},
	}

	statusErrors := []string{"compliance operator not ready.  Only 2 pods are ready when 5 are desired."}
	expectedInfo := &storage.ComplianceIntegration{
		Version:      "22",
		ClusterId:    fixtureconsts.Cluster1,
		Namespace:    fixtureconsts.Namespace1,
		StatusErrors: statusErrors,
	}

	suite.manager.EXPECT().ProcessComplianceOperatorInfo(gomock.Any(), expectedInfo).Return(nil).Times(1)

	err := suite.pipeline.Run(context.Background(), fixtureconsts.Cluster1, &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_ComplianceOperatorInfo{
			ComplianceOperatorInfo: complianceMsg,
		},
	}, nil)
	suite.NoError(err)
}
