package nodeinventory

import (
	"context"
	"testing"

	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	nodesEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
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
			name: "when event action is REMOVE_RESOURCE then ignore event",
			setUp: func(t *testing.T, a *args, m *mocks) {
				a.msg = createMsg("foobar")
				a.msg.GetEvent().Action = central.ResourceAction_REMOVE_RESOURCE
				a.injector = &recordingInjector{}
			},
			wantInjectorContain: []*central.NodeInventoryACK{},
		},
		{
			name: "when LegacyScanner is disabled then ACK and discard",
			setUp: func(t *testing.T, a *args, m *mocks) {
				testutils.MustUpdateFeature(t, features.LegacyScanner, false)
				a.msg = createMsg("node1")
				a.msg.GetEvent().GetNodeInventory().NodeName = "test-node"
				a.injector = &recordingInjector{}
			},
			wantInjectorContain: []*central.NodeInventoryACK{{Action: central.NodeInventoryACK_ACK, NodeName: "test-node"}},
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
			if tt.args.injector != nil {
				inj := tt.args.injector.(*recordingInjector)
				if len(tt.wantInjectorContain) == 0 {
					assert.Len(t, inj.getSentACKs(), 0)
				} else {
					protoassert.SlicesEqual(t, tt.wantInjectorContain, inj.getSentACKs(), "sent ACKs: %v", inj.getSentACKs())
				}
			}
		})
	}
}

var _ common.MessageInjector = (*recordingInjector)(nil)

type recordingInjector struct {
	lock         sync.Mutex
	messages     []*central.NodeInventoryACK
	sensor       []*central.SensorACK
	capabilities map[centralsensor.SensorCapability]bool
}

func (r *recordingInjector) HasCapability(cap centralsensor.SensorCapability) bool {
	return r.capabilities[cap]
}

func (r *recordingInjector) InjectMessage(_ concurrency.Waitable, msg *central.MsgToSensor) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if ack := msg.GetNodeInventoryAck(); ack != nil {
		r.messages = append(r.messages, ack.CloneVT())
	}
	if ack := msg.GetSensorAck(); ack != nil {
		r.sensor = append(r.sensor, ack.CloneVT())
	}
	return nil
}

func (r *recordingInjector) InjectMessageIntoQueue(_ *central.MsgFromSensor) {}

func (r *recordingInjector) getSentACKs() []*central.NodeInventoryACK {
	r.lock.Lock()
	defer r.lock.Unlock()
	copied := make([]*central.NodeInventoryACK, 0, len(r.messages))
	for _, m := range r.messages {
		if m != nil {
			copied = append(copied, m)
		}
	}
	return copied
}

func (r *recordingInjector) getSentSensorACKs() []*central.SensorACK {
	r.lock.Lock()
	defer r.lock.Unlock()
	copied := make([]*central.SensorACK, 0, len(r.sensor))
	for _, m := range r.sensor {
		if m != nil {
			copied = append(copied, m)
		}
	}
	return copied
}
