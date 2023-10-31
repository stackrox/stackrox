package rolebindings

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	roleBindingMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	reprocessorMocks "github.com/stackrox/rox/central/reprocessor/mocks"
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
	rolebindings    *roleBindingMocks.MockDataStore
	riskReprocessor *reprocessorMocks.MockLoop
	pipeline        *pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.rolebindings = roleBindingMocks.NewMockDataStore(suite.mockCtrl)

	suite.pipeline = &pipelineImpl{
		clusters:        suite.clusters,
		bindings:        suite.rolebindings,
		riskReprocessor: suite.riskReprocessor,
	}
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestRoleBindingSyncResources() {
	ctx := context.Background()
	rolebinding := fixtures.GetMultipleK8sRoleBindings(1, 1)[0]

	suite.clusters.EXPECT().GetClusterName(ctx, rolebinding.GetClusterId())
	suite.rolebindings.EXPECT().UpsertRoleBinding(ctx, gomock.Any())

	err := suite.pipeline.Run(ctx, rolebinding.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     rolebinding.GetId(),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_Binding{
					Binding: rolebinding,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}
