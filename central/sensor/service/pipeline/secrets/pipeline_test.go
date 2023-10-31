package secrets

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
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

	clusters *clusterMocks.MockDataStore
	secrets  *secretMocks.MockDataStore

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.secrets = secretMocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestSecretsSyncResources() {
	ctx := context.Background()
	secret := fixtures.GetSecret()

	suite.clusters.EXPECT().GetClusterName(context.Background(), "clusterid")
	suite.secrets.EXPECT().UpsertSecret(ctx, secret).Return(nil)

	pipeline := NewPipeline(suite.clusters, suite.secrets)
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     "secretid",
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_Secret{
					Secret: secret,
				},
			},
		},
	}
	err := pipeline.Run(ctx, "clusterid", msg, nil)
	suite.NoError(err)

}

func (suite *PipelineTestSuite) TestRun() {
	ctx := context.Background()
	secret := fixtures.GetSecret()
	suite.clusters.EXPECT().GetClusterName(ctx, "clusterid").Return("clustername", true, nil)
	suite.secrets.EXPECT().UpsertSecret(ctx, secret).Return(nil)

	pipeline := NewPipeline(suite.clusters, suite.secrets)
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     "secretid",
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_Secret{
					Secret: secret,
				},
			},
		},
	}
	err := pipeline.Run(ctx, "clusterid", msg, nil)
	suite.NoError(err)
}
