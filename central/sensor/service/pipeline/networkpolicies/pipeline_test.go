package networkpolicies

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	networkPolicyMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	graphMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
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
	networkPolicies *networkPolicyMocks.MockDataStore
	graphEvaluator  *graphMocks.MockEvaluator
	pipeline        *pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.networkPolicies = networkPolicyMocks.NewMockDataStore(suite.mockCtrl)
	suite.graphEvaluator = graphMocks.NewMockEvaluator(suite.mockCtrl)

	suite.pipeline =
		NewPipeline(
			suite.clusters,
			suite.networkPolicies,
			suite.graphEvaluator).(*pipelineImpl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestRoleBindingSyncResources() {
	ctx := context.Background()
	networkPolicy := fixtures.GetNetworkPolicy()

	suite.clusters.EXPECT().GetClusterName(ctx, networkPolicy.GetClusterId())
	suite.networkPolicies.EXPECT().UpsertNetworkPolicy(ctx, gomock.Any())
	suite.graphEvaluator.EXPECT().IncrementEpoch(gomock.Any()).Return()

	err := suite.pipeline.Run(ctx, networkPolicy.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     networkPolicy.GetId(),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_NetworkPolicy{
					NetworkPolicy: networkPolicy,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestRoleBindingDeleteResources() {
	ctx := context.Background()
	networkPolicy := fixtures.GetNetworkPolicy()

	suite.networkPolicies.EXPECT().RemoveNetworkPolicy(ctx, networkPolicy.GetId())
	suite.graphEvaluator.EXPECT().IncrementEpoch(networkPolicy.GetClusterId())

	err := suite.pipeline.Run(ctx, networkPolicy.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     networkPolicy.GetId(),
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_NetworkPolicy{
					NetworkPolicy: networkPolicy,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}
