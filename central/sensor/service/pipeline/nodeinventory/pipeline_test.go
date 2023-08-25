package nodeinventory

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	nodesEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_pipelineImpl_Run(t *testing.T) {
	createMsg := func(id string) *central.MsgFromSensor {
		return &central.MsgFromSensor{
			Msg: &central.MsgFromSensor_Event{
				Event: &central.SensorEvent{
					Resource: &central.SensorEvent_NodeInventory{
						NodeInventory: &storage.NodeInventory{
							NodeId: id,
						},
					},
				},
			},
		}
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
		name                string
		mocks               mocks
		args                args
		wantErr             string
		wantInjectorContain []*central.NodeInventoryACK
		setUp               func(t *testing.T, a *args, m *mocks)
	}{
		{
			name:    "when event has no node inventory then error",
			wantErr: "unexpected resource type",
		},
		{
			name: "when event action is not UNSET then ignore event",
			setUp: func(t *testing.T, a *args, m *mocks) {
				a.msg = createMsg("foobar")
				a.msg.GetEvent().Action = central.ResourceAction_CREATE_RESOURCE
			},
		},
		{
			name: "when event has inventory then enrich and upsert with risk",
			wantInjectorContain: []*central.NodeInventoryACK{
				{Action: central.NodeInventoryACK_ACK},
			},
			setUp: func(t *testing.T, a *args, m *mocks) {
				node := storage.Node{
					Id: "test node id",
				}
				a.msg = createMsg(node.GetId())
				a.injector = &recordingInjector{}
				gomock.InOrder(
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(node.GetId())).Times(1).Return(&node, true, nil),
					m.enricher.EXPECT().EnrichNodeWithInventory(gomock.Any(), gomock.Any()).Times(1).Return(nil),
					m.riskManager.EXPECT().CalculateRiskAndUpsertNode(gomock.Any()).Times(1).Return(nil),
				)
			},
		},
		{
			name: "when injector is nil then handle normally and don't panic",
			setUp: func(t *testing.T, a *args, m *mocks) {
				node := storage.Node{
					Id: "test node id",
				}
				a.msg = createMsg(node.GetId())
				a.injector = nil
				gomock.InOrder(
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(node.GetId())).Times(1).Return(&node, true, nil),
					m.enricher.EXPECT().EnrichNodeWithInventory(gomock.Any(), gomock.Any()).Times(1).Return(nil),
					m.riskManager.EXPECT().CalculateRiskAndUpsertNode(gomock.Any()).Times(1).Return(nil),
				)
			},
		},
		{
			name:                "when event has inventory for unknown node then no ACK should be sent",
			wantInjectorContain: []*central.NodeInventoryACK{},
			setUp: func(t *testing.T, a *args, m *mocks) {
				a.msg = createMsg("node1")
				a.injector = &recordingInjector{}
				gomock.InOrder(
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq("node1")).Times(1).Return(nil, false, nil),
				)
			},
		},
		{
			name:                "when fetching node errors then no ACK should be sent",
			wantInjectorContain: []*central.NodeInventoryACK{},
			wantErr:             "fetching error from DB",
			setUp: func(t *testing.T, a *args, m *mocks) {
				a.msg = createMsg("node1")
				a.injector = &recordingInjector{}
				gomock.InOrder(
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq("node1")).Times(1).Return(nil, false, errors.New("fetching error from DB")),
				)
			},
		},
	}
	for _, tt := range tests {
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
			if tt.wantInjectorContain != nil {
				inj := tt.args.injector.(*recordingInjector)
				assert.Equal(t, tt.wantInjectorContain, inj.getSentACKs())
			}
		})
	}
}

var _ common.MessageInjector = (*recordingInjector)(nil)

type recordingInjector struct {
	lock     sync.Mutex
	messages []*central.NodeInventoryACK
}

func (r *recordingInjector) InjectMessage(_ concurrency.Waitable, msg *central.MsgToSensor) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.messages = append(r.messages, msg.GetNodeInventoryAck().Clone())
	return nil
}

func (r *recordingInjector) InjectMessageIntoQueue(_ *central.MsgFromSensor) {}

func (r *recordingInjector) getSentACKs() []*central.NodeInventoryACK {
	r.lock.Lock()
	defer r.lock.Unlock()
	copied := make([]*central.NodeInventoryACK, 0, len(r.messages))
	copied = append(copied, r.messages...)
	return copied
}
