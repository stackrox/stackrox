package nodes

import (
	"context"
	"testing"

	protobuf "github.com/gogo/protobuf/types"
	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	nodesEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	clusterStore  *clusterDatastoreMocks.MockDataStore
	nodeDatastore *nodeDatastoreMocks.MockDataStore
	enricher      *nodesEnricherMocks.MockNodeEnricher
	riskManager   *riskManagerMocks.MockManager
	pipeline      *pipelineImpl

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.clusterStore = clusterDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.nodeDatastore = nodeDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.enricher = nodesEnricherMocks.NewMockNodeEnricher(suite.mockCtrl)
	suite.riskManager = riskManagerMocks.NewMockManager(suite.mockCtrl)

	suite.pipeline = &pipelineImpl{
		clusterStore:  suite.clusterStore,
		nodeDatastore: suite.nodeDatastore,
		enricher:      suite.enricher,
		riskManager:   suite.riskManager,
	}
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *PipelineTestSuite) TestNodeSyncResources() {
	ctx := context.Background()
	node := fixtures.GetNode()

	suite.clusterStore.EXPECT().GetClusterName(ctx, node.GetClusterId())
	suite.enricher.EXPECT().EnrichNode(node)
	suite.riskManager.EXPECT().CalculateRiskAndUpsertNode(node)

	err := suite.pipeline.Run(ctx, node.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     node.GetId(),
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_Node{
					Node: node,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}

func (suite *PipelineTestSuite) TestNodeDeleteResources() {
	ctx := context.Background()
	node := fixtures.GetNode()

	suite.nodeDatastore.EXPECT().DeleteNodes(ctx, node.GetId())

	err := suite.pipeline.Run(ctx, node.GetClusterId(), &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id:     node.GetId(),
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_Node{
					Node: node,
				},
			},
		},
	}, nil)
	suite.NoError(err)
}

func Test_pipelineImpl_Run(t *testing.T) {
	createMsg := func(osImage string) *central.MsgFromSensor {
		return &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Resource: &central.SensorEvent_Node{Node: &storage.Node{
						OsImage: osImage,
						// Set timestamp to assert it was nilled later on.
						LastUpdated: protobuf.TimestampNow(),
					}}},
			}}
	}
	type mocks struct {
		clusterStore  *clusterDatastoreMocks.MockDataStore
		nodeDatastore *nodeDatastoreMocks.MockDataStore
		riskManager   *riskManagerMocks.MockManager
		enricher      *nodesEnricherMocks.MockNodeEnricher
	}
	type args struct {
		ctx       context.Context
		clusterID string
		msg       *central.MsgFromSensor
		injector  common.MessageInjector
	}
	tests := []struct {
		name    string
		mocks   mocks
		args    args
		wantErr string
		setUp   func(t *testing.T, a *args, m *mocks)
	}{
		{
			name:    "when event has no node then error",
			wantErr: "unexpected resource type",
		},
		{
			name: "when node is full host scanned then no enrich and upsert without risk",
			setUp: func(t *testing.T, a *args, m *mocks) {
				a.msg = createMsg("Red Hat Enterprise Linux CoreOS 412.86.202302091419-0 (Ootpa)")
				a.clusterID = "test cluster id"
				gomock.InOrder(
					m.clusterStore.EXPECT().
						GetClusterName(gomock.Any(), gomock.Eq(a.clusterID)).
						Times(1).
						Return("test cluster name", true, nil),
					m.nodeDatastore.EXPECT().
						UpsertNode(gomock.Any(), gomock.Any()).
						Times(1).
						DoAndReturn(func(_ context.Context, node *storage.Node) error {
							assert.Equal(t, node.ClusterName, "test cluster name")
							assert.Equal(t, node.ClusterId, a.clusterID)
							return nil
						}),
				)
			},
		},
		{
			name: "when node is not full host scanned then enrich and upsert with risk",
			setUp: func(t *testing.T, a *args, m *mocks) {
				a.msg = createMsg("Something that is not RHCOS")
				a.clusterID = "test cluster id"
				gomock.InOrder(
					m.clusterStore.EXPECT().
						GetClusterName(gomock.Any(), gomock.Eq(a.clusterID)).
						Times(1).
						Return("test cluster name", true, nil),
					m.enricher.EXPECT().
						EnrichNode(gomock.Any()).
						Times(1).
						Return(nil),
					m.riskManager.EXPECT().
						CalculateRiskAndUpsertNode(gomock.Any()).
						DoAndReturn(func(node *storage.Node) error {
							assert.Equal(t, node.ClusterName, "test cluster name")
							assert.Equal(t, node.ClusterId, a.clusterID)
							return nil
						}).Times(1),
				)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			tt.mocks = mocks{
				clusterStore:  clusterDatastoreMocks.NewMockDataStore(ctrl),
				nodeDatastore: nodeDatastoreMocks.NewMockDataStore(ctrl),
				riskManager:   riskManagerMocks.NewMockManager(ctrl),
				enricher:      nodesEnricherMocks.NewMockNodeEnricher(ctrl),
			}
			if tt.setUp != nil {
				tt.setUp(t, &tt.args, &tt.mocks)
			}
			p := &pipelineImpl{
				clusterStore:  tt.mocks.clusterStore,
				nodeDatastore: tt.mocks.nodeDatastore,
				enricher:      tt.mocks.enricher,
				riskManager:   tt.mocks.riskManager,
			}
			if err := p.Run(tt.args.ctx, tt.args.clusterID, tt.args.msg, tt.args.injector); (err != nil) != (tt.wantErr != "") {
				assert.ErrorContainsf(t, err, tt.wantErr, "Run() error = %v, wantErr = %q", err, tt.wantErr)
			}
		})
	}
}
