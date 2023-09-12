package nodeinventory

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/activecomponent/updater"
	updaterMocks "github.com/stackrox/rox/central/activecomponent/updater/mocks"
	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	nodeCVEDataStoreMocks "github.com/stackrox/rox/central/cve/node/datastore/mocks"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageStoreMock "github.com/stackrox/rox/central/image/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskStoreMock "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/nodes"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/metrics"
	nodeEnricher "github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

const (
	nodeID              = "node-id-1"
	nodeName            = "node-name"
	clusterID           = "cluster1"
	kernelComponentName = "kernel"
)

func Test_TwoPipelines_Run(t *testing.T) {
	nodeWithScore := &storage.Node{
		Id:            nodeID,
		Name:          nodeName,
		ClusterId:     clusterID,
		ClusterName:   clusterID,
		KernelVersion: "v1",
		Notes:         []storage.Node_Note{storage.Node_MISSING_SCAN_DATA},
		RiskScore:     1,
	}

	nodeWithScanWithKernelV1 := &storage.Node{
		Id:            nodeID,
		Name:          nodeName,
		ClusterId:     clusterID,
		ClusterName:   clusterID,
		KernelVersion: "v1",
		Notes:         []storage.Node_Note{},
		RiskScore:     1,
		Scan:          nodeScanFixtureWithKernel("v1"),
		SetComponents: &storage.Node_Components{Components: 1},
		SetCves:       &storage.Node_Cves{Cves: 1},
		SetTopCvss:    &storage.Node_TopCvss{TopCvss: 1},
	}

	nodeWithScanWithKernelV2 := &storage.Node{
		Id:            nodeID,
		Name:          nodeName,
		ClusterId:     clusterID,
		ClusterName:   clusterID,
		KernelVersion: "v1",
		Notes:         []storage.Node_Note{},
		RiskScore:     1,
		Scan:          nodeScanFixtureWithKernel("v2"),
		SetComponents: &storage.Node_Components{Components: 1},
		SetCves:       &storage.Node_Cves{Cves: 1},
		SetTopCvss:    &storage.Node_TopCvss{TopCvss: 1},
	}

	type usedMocks struct {
		clusterStore      *clusterDatastoreMocks.MockDataStore
		nodeDatastore     *nodeDatastoreMocks.MockDataStore
		cveDatastore      *nodeCVEDataStoreMocks.MockDataStore
		deploymentStorage datastore.DataStore
		imageStorage      imageDS.DataStore
		riskStorage       *riskStoreMock.MockDataStore
		updater           updater.Updater
	}
	tests := map[string]struct {
		mocks                     usedMocks
		riskManager               manager.Manager
		enricher                  nodeEnricher.NodeEnricher
		operations                []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error
		wantErr                   string
		setUpMocks                func(t *testing.T, m *usedMocks)
		wantNodeExists            bool
		wantKernelVersionNode     string
		wantKernelVersionNodeScan string
	}{
		"lone node inventory should not find the node in DB": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error{
				// Node-scan (node-inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error {
					return ninvp.Run(context.Background(), clusterID, createNodeInventoryMsg(nodeID, "v2"), nil)
				},
			},
			setUpMocks: func(t *testing.T, m *usedMocks) {
				m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).MinTimes(1).Return(nil, false, nil)
			},
			wantNodeExists:        false,
			wantKernelVersionNode: "",
		},
		"node inventory arriving after node should result in data from the node being overwritten": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error{
				// Old node-scan for node1 arrives over the node pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error {
					return np.Run(context.Background(), clusterID, createNodeMsg(nodeID, "v1"), nil)
				},
				// New node-scan (node-inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error {
					return ninvp.Run(context.Background(), clusterID, createNodeInventoryMsg(nodeID, "v2"), nil)
				},
			},
			setUpMocks: func(t *testing.T, m *usedMocks) {
				gomock.InOrder(
					// node arrives
					m.clusterStore.EXPECT().GetClusterName(gomock.Any(), gomock.Eq(clusterID)).Times(1).Return(clusterID, true, nil),
					m.cveDatastore.EXPECT().EnrichNodeWithSuppressedCVEs(gomock.Any()),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).AnyTimes().Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), gomock.Any()).AnyTimes().Return(nil),

					// node inventory arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).AnyTimes().Return(nodeWithScore, true, nil),
					m.cveDatastore.EXPECT().EnrichNodeWithSuppressedCVEs(gomock.Any()).AnyTimes().Return(),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).AnyTimes().Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), nodeWithScanWithKernelV2).AnyTimes().Return(nil),
					// check what got stored in the DB
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).AnyTimes().Return(nodeWithScanWithKernelV2, true, nil),
				)
			},
			wantNodeExists:            true,
			wantKernelVersionNode:     "v1",
			wantKernelVersionNodeScan: "v2",
		},
		"node inventory arriving first should result in data from it being lost": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error{
				// New node-scan (node-inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error {
					return ninvp.Run(context.Background(), clusterID, createNodeInventoryMsg(nodeID, "v2"), nil)
				},
				// Old node-scan for node1 arrives over the node pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment) error {
					return np.Run(context.Background(), clusterID, createNodeMsg(nodeID, "v1"), nil)
				},
			},
			setUpMocks: func(t *testing.T, m *usedMocks) {
				gomock.InOrder(
					// node inventory arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Return(nil, false, nil),

					// node arrives
					m.clusterStore.EXPECT().GetClusterName(gomock.Any(), gomock.Eq(clusterID)).Return(clusterID, true, nil),
					m.cveDatastore.EXPECT().EnrichNodeWithSuppressedCVEs(gomock.Any()),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).Times(2).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), gomock.Any()).Return(nil),

					// check what got stored in the DB
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).AnyTimes().Return(nodeWithScanWithKernelV1, true, nil),
				)
			},
			wantNodeExists:            true,
			wantKernelVersionNode:     "v1",
			wantKernelVersionNodeScan: "v1",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			tt.mocks = usedMocks{
				clusterStore:      clusterDatastoreMocks.NewMockDataStore(ctrl),
				nodeDatastore:     nodeDatastoreMocks.NewMockDataStore(ctrl),
				cveDatastore:      nodeCVEDataStoreMocks.NewMockDataStore(ctrl),
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
			creator := func() (string, scanners.NodeScannerCreator) {
				return "fake", func(integration *storage.NodeIntegration) (types.NodeScanner, error) {
					return &fakeNodeScanner{}, nil
				}
			}
			tt.enricher = nodeEnricher.NewWithCreator(tt.mocks.cveDatastore, metrics.CentralSubsystem, creator)
			err := tt.enricher.UpsertNodeIntegration(&storage.NodeIntegration{
				Id:   "1",
				Name: "dummy-scanner",
				Type: "fake",
				IntegrationConfig: &storage.NodeIntegration_Clairify{Clairify: &storage.ClairifyConfig{
					Endpoint:           "abc",
					GrpcEndpoint:       "",
					NumConcurrentScans: 0,
				}},
			})
			assert.NoError(t, err)

			if tt.setUpMocks != nil {
				tt.setUpMocks(t, &tt.mocks)
			}
			pNode := nodes.NewPipeline(tt.mocks.clusterStore, tt.mocks.nodeDatastore, tt.enricher, tt.riskManager)
			pNodeInv := newPipeline(tt.mocks.clusterStore, tt.mocks.nodeDatastore, tt.enricher, tt.riskManager)

			var lastErr error
			for i, op := range tt.operations {
				t.Logf("Running operation %d of %d", i+1, len(tt.operations))
				lastErr = op(t, pNode, pNodeInv)
			}
			if tt.wantErr != "" {
				assert.ErrorContainsf(t, lastErr, tt.wantErr, "Run() error = %v, wantErr = %q", lastErr, tt.wantErr)
			}

			node, found, err := tt.mocks.nodeDatastore.GetNode(context.Background(), nodeID)
			assert.Equal(t, tt.wantNodeExists, found)
			assert.NoError(t, err)
			if found {
				assert.Equal(t, tt.wantKernelVersionNode, node.GetKernelVersion())
				var kernelComponentFound bool
				for _, component := range node.GetScan().GetComponents() {
					if component.GetName() == kernelComponentName {
						kernelComponentFound = true
						assert.Equal(t, tt.wantKernelVersionNodeScan, component.GetVersion(), "kernel version in node scan should match")
					}
				}
				assert.True(t, kernelComponentFound)

			}
		})
	}
}

func createNodeInventory(id, kernelV string) *storage.NodeInventory {
	return &storage.NodeInventory{
		NodeId:   id,
		NodeName: nodeName,
		Components: &storage.NodeInventory_Components{
			Namespace: "",
			RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
				{
					Id:          1,
					Name:        kernelComponentName,
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
						Name:          nodeName,
					},
				},
			},
		},
	}
}

func nodeScanFixtureWithKernel(kernelVersion string) *storage.NodeScan {
	return &storage.NodeScan{
		Components: []*storage.EmbeddedNodeScanComponent{
			{
				Name:    kernelComponentName,
				Version: kernelVersion,
				Vulns:   nil,
				Vulnerabilities: []*storage.NodeVulnerability{
					{
						CveBaseInfo: &storage.CVEInfo{
							Cve: "CVE-2020-1234",
						},
						Cvss:         1,
						Severity:     0,
						SetFixedBy:   nil,
						Snoozed:      false,
						SnoozeStart:  nil,
						SnoozeExpiry: nil,
					},
				},
				Priority:   0,
				SetTopCvss: &storage.EmbeddedNodeScanComponent_TopCvss{TopCvss: 1.0},
				RiskScore:  1,
			},
		},
	}
}

var _ types.NodeScanner = (*fakeNodeScanner)(nil)

type fakeNodeScanner struct {
	requestedScan bool
}

func (*fakeNodeScanner) MaxConcurrentNodeScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f *fakeNodeScanner) GetNodeScan(*storage.Node) (*storage.NodeScan, error) {
	f.requestedScan = true
	return nodeScanFixtureWithKernel("v1"), nil
}

func (f *fakeNodeScanner) GetNodeInventoryScan(*storage.Node, *storage.NodeInventory) (*storage.NodeScan, error) {
	f.requestedScan = true
	return nodeScanFixtureWithKernel("v2"), nil
}

func (*fakeNodeScanner) TestNodeScanner() error {
	return nil
}

func (*fakeNodeScanner) Type() string {
	return "type"
}

func (*fakeNodeScanner) Name() string {
	return "name"
}

func getDummyRisk() *storage.Risk {
	return &storage.Risk{
		Score:   1.0,
		Results: make([]*storage.Risk_Result, 0),
		Subject: &storage.RiskSubject{},
	}
}

type mockNodeScorer struct{}

func (m *mockNodeScorer) Score(_ context.Context, _ *storage.Node) *storage.Risk {
	return getDummyRisk()
}

type mockComponentScorer struct{}

func (m *mockComponentScorer) Score(_ context.Context, _ scancomponent.ScanComponent, _ string) *storage.Risk {
	return getDummyRisk()
}

type mockDeploymentScorer struct{}

func (m *mockDeploymentScorer) Score(_ context.Context, _ *storage.Deployment, _ []*storage.Risk) *storage.Risk {
	return getDummyRisk()
}

type mockImageScorer struct{}

func (m *mockImageScorer) Score(_ context.Context, _ *storage.Image) *storage.Risk {
	return getDummyRisk()
}
