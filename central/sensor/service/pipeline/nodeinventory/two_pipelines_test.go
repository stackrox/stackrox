package nodeinventory

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/activecomponent/updater"
	updaterMocks "github.com/stackrox/rox/central/activecomponent/updater/mocks"
	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageStoreMock "github.com/stackrox/rox/central/image/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskStoreMock "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/nodes"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	nodesEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_TwoPipelines_Run(t *testing.T) {
	const nodeID = "node-id-1"

	type usedMocks struct {
		clusterStore      *clusterDatastoreMocks.MockDataStore
		nodeDatastore     *nodeDatastoreMocks.MockDataStore
		enricher          *nodesEnricherMocks.MockNodeEnricher
		deploymentStorage datastore.DataStore
		imageStorage      imageDS.DataStore
		riskStorage       *riskStoreMock.MockDataStore
		updater           updater.Updater
	}
	type args struct {
		ctx       context.Context
		clusterID string
		injector  common.MessageInjector
	}
	tests := []struct {
		name                string
		mocks               usedMocks
		riskManager         manager.Manager
		args                args
		operations          []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, args args) error
		wantErr             string
		wantInjectorContain []*central.NodeInventoryACK
		wantKernel          string
		setUpMocks          func(t *testing.T, a *args, m *usedMocks)
	}{
		{
			name: "node inventory arriving after node should result in data from the node being overwritten",
			args: args{
				ctx:       context.Background(),
				clusterID: "cluster1",
				injector:  nil,
			},
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, args args) error{
				// Old node-scan for node1 arrives over the node pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, args args) error {
					return np.Run(args.ctx, args.clusterID, createNodeMsg(nodeID, "v1"), args.injector)
				},
				// New node-scan (node-inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, args args) error {
					return ninvp.Run(args.ctx, args.clusterID, createNodeInventoryMsg(nodeID, "v2"), args.injector)
				},
			},
			setUpMocks: func(t *testing.T, a *args, m *usedMocks) {
				node1 := &storage.Node{
					Id:            nodeID,
					ClusterId:     a.clusterID,
					ClusterName:   a.clusterID,
					KernelVersion: "v1",
				}
				nInv := createNodeInventory(nodeID, "v2")
				gomock.InOrder(
					m.clusterStore.EXPECT().GetClusterName(gomock.Any(), gomock.Eq(a.clusterID)).Times(1).Return(a.clusterID, true, nil),
					m.enricher.EXPECT().EnrichNode(gomock.Any()).Times(1).Return(nil),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).Times(1).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), gomock.Any()).Times(1).Return(nil),

					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(node1, true, nil),
					m.enricher.EXPECT().EnrichNodeWithInventory(node1, nInv).Times(1).Return(nil),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).Times(1).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), gomock.Any()).Times(1).Return(nil),
				)
			},
			wantKernel: "v2",
		},
		{
			name: "node inventory arriving first should result in data from it being lost",
			args: args{
				ctx:       context.Background(),
				clusterID: "cluster1",
				injector:  nil,
			},
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, args args) error{
				// New node-scan (node-inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, args args) error {
					return ninvp.Run(args.ctx, args.clusterID, createNodeInventoryMsg(nodeID, "v2"), args.injector)
				},
				// Old node-scan for node1 arrives over the node pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, args args) error {
					return np.Run(args.ctx, args.clusterID, createNodeMsg(nodeID, "v1"), args.injector)
				},
			},
			setUpMocks: func(t *testing.T, a *args, m *usedMocks) {
				gomock.InOrder(
					// node inventory will be ignored - false means node nod found
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nil, false, nil),

					// node will be hadled normally
					m.clusterStore.EXPECT().GetClusterName(gomock.Any(), gomock.Eq(a.clusterID)).Times(1).Return(a.clusterID, true, nil),
					m.enricher.EXPECT().EnrichNode(gomock.Any()).Times(1).Return(nil),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).Times(1).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), gomock.Any()).Times(1).Return(nil),
				)
			},
			wantKernel: "v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			tt.mocks = usedMocks{
				clusterStore:      clusterDatastoreMocks.NewMockDataStore(ctrl),
				nodeDatastore:     nodeDatastoreMocks.NewMockDataStore(ctrl),
				enricher:          nodesEnricherMocks.NewMockNodeEnricher(ctrl),
				deploymentStorage: mocks.NewMockDataStore(ctrl),
				imageStorage:      imageStoreMock.NewMockDataStore(ctrl),
				riskStorage:       riskStoreMock.NewMockDataStore(ctrl),
				updater:           updaterMocks.NewMockUpdater(ctrl),
			}
			tt.riskManager = manager.New(
				tt.mocks.nodeDatastore,
				tt.mocks.deploymentStorage,
				tt.mocks.imageStorage,
				tt.mocks.riskStorage,
				&mockNodeScorer{},
				&mockComponentScorer{},
				&mockDeploymentScorer{},
				&mockImageScorer{},
				&mockComponentScorer{},

				ranking.ClusterRanker(),
				ranking.NamespaceRanker(),
				ranking.ComponentRanker(),
				ranking.NodeComponentRanker(),

				tt.mocks.updater,
			)
			if tt.setUpMocks != nil {
				tt.setUpMocks(t, &tt.args, &tt.mocks)
			}
			pNode := nodes.NewPipeline(tt.mocks.clusterStore, tt.mocks.nodeDatastore, tt.mocks.enricher, tt.riskManager)
			pNodeInv := newPipeline(tt.mocks.clusterStore, tt.mocks.nodeDatastore, tt.mocks.enricher, tt.riskManager)

			var lastErr error
			for i, op := range tt.operations {
				t.Logf("Running operation %d of %d", i+1, len(tt.operations))
				lastErr = op(t, pNode, pNodeInv, tt.args)
			}
			if tt.wantErr != "" {
				assert.ErrorContainsf(t, lastErr, tt.wantErr, "Run() error = %v, wantErr = %q", lastErr, tt.wantErr)
			}
			if tt.wantInjectorContain != nil {
				inj := tt.args.injector.(*recordingInjector)
				assert.Equal(t, tt.wantInjectorContain, inj.getSentACKs())
			}

			// FIXME: work on this assertion
			t.Logf("want kernel: %s, got kernel: %s", tt.wantKernel, "????????")
		})
	}
}

func createNodeInventory(id, kernelV string) *storage.NodeInventory {
	return &storage.NodeInventory{
		NodeId: id,
		Components: &storage.NodeInventory_Components{
			Namespace: "",
			RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
				{
					Id:          1,
					Name:        "kernel",
					Namespace:   "",
					Version:     kernelV,
					Arch:        "",
					Module:      "",
					AddedBy:     "",
					Executables: nil,
				},
			},
			RhelContentSets: nil,
		},
	}
}

func createNodeInventoryMsg(id, kernel string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_NodeInventory{
					NodeInventory: createNodeInventory(id, kernel),
				},
			},
		},
	}
}

func createNodeMsg(id, kernel string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Node{
					Node: &storage.Node{
						Id:            id,
						KernelVersion: kernel,
					},
				},
			},
		},
	}
}
