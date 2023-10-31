package roles

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
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
	roles           *roleMocks.MockDataStore
	riskReprocessor *reprocessorMocks.MockLoop
	pipeline        *pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.roles = roleMocks.NewMockDataStore(suite.mockCtrl)
	suite.riskReprocessor = reprocessorMocks.NewMockLoop(suite.mockCtrl)

	suite.pipeline = &pipelineImpl{
		clusters:        suite.clusters,
		roles:           suite.roles,
		riskReprocessor: suite.riskReprocessor,
	}
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestRoleSyncResources() {
	ctx := context.Background()
	role := fixtures.GetMultipleK8SRoles(1)[0]

	suite.clusters.EXPECT().GetClusterName(ctx, role.GetId())
	suite.roles.EXPECT().UpsertRole(ctx, gomock.Any())

	err := suite.pipeline.Run(ctx, role.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     role.GetId(),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_Role{
					Role: role,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}
