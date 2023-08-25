package auditlogstateupdate

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	clusters *clusterMocks.MockDataStore

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.clusters = clusterMocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestRun() {
	ctx := context.Background()
	statusInfo := &central.AuditLogStatusInfo{
		NodeAuditLogFileStates: map[string]*storage.AuditLogFileState{
			"node-a": {
				CollectLogsSince: types.TimestampNow(),
				LastAuditId:      "last-id",
			},
		},
	}
	suite.clusters.EXPECT().UpdateAuditLogFileStates(ctx, "clusterid", statusInfo.GetNodeAuditLogFileStates()).Return(nil)

	pipeline := NewPipeline(suite.clusters)
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_AuditLogStatusInfo{
			AuditLogStatusInfo: statusInfo,
		},
	}
	suite.NoError(pipeline.Run(ctx, "clusterid", msg, nil))
}

func (suite *PipelineTestSuite) TestMatchOnlyMatchesAuditLogStatusInfoMsg() {
	pipeline := NewPipeline(suite.clusters)

	statusMsg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_AuditLogStatusInfo{
			AuditLogStatusInfo: &central.AuditLogStatusInfo{},
		},
	}
	suite.True(pipeline.Match(statusMsg), "When given AuditLogStatusInfo it should match")

	helloMsg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Hello{
			Hello: &central.SensorHello{},
		},
	}
	suite.False(pipeline.Match(helloMsg), "When given another message it should _not_ match")
}
