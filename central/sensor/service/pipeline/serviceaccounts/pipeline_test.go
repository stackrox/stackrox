package serviceaccounts

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	reprocessorMocks "github.com/stackrox/rox/central/reprocessor/mocks"
	serviceAccountMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"

	"go.uber.org/mock/gomock"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	clusters        *clusterMocks.MockDataStore
	serviceaccounts *serviceAccountMocks.MockDataStore
	deployments     *deploymentMocks.MockDataStore
	riskReprocessor *reprocessorMocks.MockLoop
	pipeline        *pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.serviceaccounts = serviceAccountMocks.NewMockDataStore(suite.mockCtrl)
	suite.riskReprocessor = reprocessorMocks.NewMockLoop(suite.mockCtrl)

	suite.pipeline = &pipelineImpl{
		clusters:             suite.clusters,
		deployments:          suite.deployments,
		serviceaccounts:      suite.serviceaccounts,
		riskReprocessor:      suite.riskReprocessor,
		reconciliationSignal: concurrency.NewSignal(),
	}
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestServiceAccountSyncResources() {
	ctx := context.Background()
	serviceAccount := fixtures.GetServiceAccount()

	suite.clusters.EXPECT().GetClusterName(ctx, serviceAccount.GetClusterId())
	suite.serviceaccounts.EXPECT().UpsertServiceAccount(ctx, gomock.Any())

	err := suite.pipeline.Run(ctx, serviceAccount.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     serviceAccount.GetClusterId(),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_ServiceAccount{
					ServiceAccount: serviceAccount,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestServiceAccountDeleteResources() {
	ctx := context.Background()
	serviceAccount := fixtures.GetServiceAccount()

	suite.serviceaccounts.EXPECT().RemoveServiceAccount(ctx, serviceAccount.GetId())

	err := suite.pipeline.Run(ctx, serviceAccount.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     serviceAccount.GetClusterId(),
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_ServiceAccount{
					ServiceAccount: serviceAccount,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}
