package nodeindex

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
	"github.com/stackrox/rox/central/sensor/service/pipeline/nodeinventory"
	"github.com/stackrox/rox/central/sensor/service/pipeline/nodes"
	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/metrics"
	nodeEnricher "github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

const (
	nodeID              = "node-id-1"
	nodeName            = "node-name"
	clusterID           = "cluster1"
	kernelComponentName = "kernel"
)

func Test_ThreePipelines_Run(t *testing.T) {
	nodeWithScore := &storage.Node{
		Id:            nodeID,
		Name:          nodeName,
		ClusterId:     clusterID,
		ClusterName:   clusterID,
		KernelVersion: "v1",
		Notes:         []storage.Node_Note{storage.Node_MISSING_SCAN_DATA},
		RiskScore:     1,
	}

	nodeWithV4Scan := &storage.Node{
		Id:            nodeID,
		Name:          nodeName,
		ClusterId:     clusterID,
		ClusterName:   clusterID,
		KernelVersion: "v1",
		Notes:         []storage.Node_Note{storage.Node_MISSING_SCAN_DATA},
		RiskScore:     1,
		Scan:          &storage.NodeScan{ScannerVersion: storage.NodeScan_SCANNER_V4},
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

	nodeWithScanWithKernelV4 := &storage.Node{
		Id:            nodeID,
		Name:          nodeName,
		ClusterId:     clusterID,
		ClusterName:   clusterID,
		KernelVersion: "v1",
		Notes:         []storage.Node_Note{},
		RiskScore:     1,
		Scan:          nodeScanFixtureWithKernel("v4"),
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
		mocks                     *usedMocks
		riskManager               manager.Manager
		enricher                  nodeEnricher.NodeEnricher
		operations                []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error
		setUpMocksAndEnv          func(t *testing.T, m *usedMocks)
		wantNodeExists            bool
		wantKernelVersionNode     string
		wantKernelVersionNodeScan string
	}{
		"lone node index should not find the node in the DB": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error{
				// V4 node-scan (node index) for node1 arrives
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return nidxp.Run(context.Background(), clusterID, createNodeIndexMsg(nodeID, "v3"), nil)
				},
			},
			setUpMocksAndEnv: func(t *testing.T, m *usedMocks) {
				t.Setenv(features.ScannerV4.EnvVar(), "true")
				t.Setenv(features.NodeIndexEnabled.EnvVar(), "true")
				gomock.InOrder(
					// node index arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).MinTimes(1).Return(nil, false, nil),
				)
			},
			wantNodeExists: false,
		},

		"node index arriving after node should result in data from the node being overwritten": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error{
				// V1 node-scan for node1 arrives over the node pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return np.Run(context.Background(), clusterID, createNodeMsg(nodeID, "v1"), nil)
				},
				// V4 node-scan (node index) for node1 arrives over the node index pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return nidxp.Run(context.Background(), clusterID, createNodeIndexMsg(nodeID, "v4"), nil)
				},
			},
			setUpMocksAndEnv: func(t *testing.T, m *usedMocks) {
				t.Setenv(features.ScannerV4.EnvVar(), "true")
				t.Setenv(features.NodeIndexEnabled.EnvVar(), "true")
				gomock.InOrder(
					// node arrives
					m.clusterStore.EXPECT().GetClusterName(gomock.Any(), gomock.Eq(clusterID)).Times(1).Return(clusterID, true, nil),
					m.cveDatastore.EXPECT().EnrichNodeWithSuppressedCVEs(gomock.Any()).Times(1).Return(),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).MinTimes(1).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), gomock.Any()).Times(1).Return(nil),
					// node index arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nodeWithScore, true, nil),
					m.cveDatastore.EXPECT().EnrichNodeWithSuppressedCVEs(gomock.Any()).Times(1).Return(),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).MinTimes(1).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), nodeWithScanWithKernelV4).Times(1).Return(nil),
					// check what got stored in the DB
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nodeWithScanWithKernelV4, true, nil),
				)
			},
			wantNodeExists:            true,
			wantKernelVersionNode:     "v1",
			wantKernelVersionNodeScan: "v4",
		},

		"node index arriving after node inventory should result in inventory being overwritten": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error{

				// V2 node-scan (node inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return ninvp.Run(context.Background(), clusterID, createNodeInventoryMsg(nodeID, "v2"), nil)
				},
				// V4 node-scan (node index) for node1 arrives
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return nidxp.Run(context.Background(), clusterID, createNodeIndexMsg(nodeID, "v3"), nil)
				},
			},
			setUpMocksAndEnv: func(t *testing.T, m *usedMocks) {
				t.Setenv(features.ScannerV4.EnvVar(), "true")
				t.Setenv(features.NodeIndexEnabled.EnvVar(), "true")
				gomock.InOrder(
					// node inventory arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nodeWithScore, true, nil),
					m.cveDatastore.EXPECT().EnrichNodeWithSuppressedCVEs(gomock.Any()).Times(1).Return(),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).MinTimes(1).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), nodeWithScanWithKernelV2).Times(1).Return(nil),

					// node index arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nodeWithScore, true, nil),
					m.cveDatastore.EXPECT().EnrichNodeWithSuppressedCVEs(gomock.Any()).Times(1).Return(),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).MinTimes(1).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), nodeWithScanWithKernelV4).Times(1).Return(nil),

					// check what got stored in the DB
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nodeWithScanWithKernelV4, true, nil),
				)
			},
			wantNodeExists:            true,
			wantKernelVersionNode:     "v1",
			wantKernelVersionNodeScan: "v4",
		},

		"node index arriving before node inventory should result in inventory being discarded": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error{
				// V4 node-scan (node index) for node1 arrives
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return nidxp.Run(context.Background(), clusterID, createNodeIndexMsg(nodeID, "v4"), nil)
				},
				// V2 node-scan (node inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return ninvp.Run(context.Background(), clusterID, createNodeInventoryMsg(nodeID, "v2"), nil)
				},
			},
			setUpMocksAndEnv: func(t *testing.T, m *usedMocks) {
				t.Setenv(features.ScannerV4.EnvVar(), "true")
				t.Setenv(features.NodeIndexEnabled.EnvVar(), "true")
				gomock.InOrder(
					// node index arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nodeWithScore, true, nil),
					m.cveDatastore.EXPECT().EnrichNodeWithSuppressedCVEs(gomock.Any()).Times(1).Return(),
					m.riskStorage.EXPECT().UpsertRisk(gomock.Any(), gomock.Any()).MinTimes(1).Return(nil),
					m.nodeDatastore.EXPECT().UpsertNode(gomock.Any(), nodeWithScanWithKernelV4).Times(1).Return(nil),

					// node inventory arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nodeWithV4Scan, true, nil),

					// check what got stored in the DB
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Times(1).Return(nodeWithScanWithKernelV4, true, nil),
				)
			},
			wantNodeExists:            true,
			wantKernelVersionNode:     "v1",
			wantKernelVersionNodeScan: "v4",
		},

		"node inventory and node index arriving while indexing is disabled results in inventory being persisted": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error{
				// V2 node-scan (node inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return ninvp.Run(context.Background(), clusterID, createNodeInventoryMsg(nodeID, "v2"), nil)
				},
				// V4 node-scan (node index) for node1 arrives
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return nidxp.Run(context.Background(), clusterID, createNodeIndexMsg(nodeID, "v4"), nil)
				},
			},
			setUpMocksAndEnv: func(t *testing.T, m *usedMocks) {
				t.Setenv(features.ScannerV4.EnvVar(), "false")
				t.Setenv(features.NodeIndexEnabled.EnvVar(), "false")
				gomock.InOrder(
					// node inventory arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Return(nodeWithScore, true, nil),
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
		"node inventory arriving on blank node while indexing is enabled still results in inventory being persisted": {
			operations: []func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error{
				// V2 node-scan (node inventory) for node1 arrives over the node-inventory pipeline
				func(t *testing.T, np pipeline.Fragment, ninvp pipeline.Fragment, nidxp pipeline.Fragment) error {
					return ninvp.Run(context.Background(), clusterID, createNodeInventoryMsg(nodeID, "v2"), nil)
				},
			},
			setUpMocksAndEnv: func(t *testing.T, m *usedMocks) {
				t.Setenv(features.ScannerV4.EnvVar(), "true")
				t.Setenv(features.NodeIndexEnabled.EnvVar(), "true")
				gomock.InOrder(
					// node inventory arrives
					m.nodeDatastore.EXPECT().GetNode(gomock.Any(), gomock.Eq(nodeID)).Return(nodeWithScore, true, nil),
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
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			tt.mocks = &usedMocks{
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

				nil,
			)
			// The test requires both, a configured Scanner v2 and v4 node integration
			creator := func() (string, scanners.NodeScannerCreator) {
				return types.Clairify, func(integration *storage.NodeIntegration) (types.NodeScanner, error) {
					return &fakeNodeScanner{}, nil
				}
			}
			creatorV4 := func() (string, scanners.NodeScannerCreator) {
				return types.ScannerV4, func(integration *storage.NodeIntegration) (types.NodeScanner, error) {
					return &fakeNodeScannerv4{}, nil
				}
			}
			tt.enricher = nodeEnricher.NewWithCreator(tt.mocks.cveDatastore, metrics.CentralSubsystem, creator, creatorV4)
			err := tt.enricher.UpsertNodeIntegration(&storage.NodeIntegration{
				Id:   "1",
				Name: "dummy-scanner",
				Type: types.Clairify,
				IntegrationConfig: &storage.NodeIntegration_Clairify{Clairify: &storage.ClairifyConfig{
					Endpoint:           "abc",
					GrpcEndpoint:       "",
					NumConcurrentScans: 0,
				}},
			})
			require.NoError(t, err)
			err = tt.enricher.UpsertNodeIntegration(&storage.NodeIntegration{
				Id:   "2",
				Name: "dummy-scanner-v4",
				Type: types.ScannerV4,
				IntegrationConfig: &storage.NodeIntegration_Scannerv4{Scannerv4: &storage.ScannerV4Config{
					NumConcurrentScans: 0,
					IndexerEndpoint:    "",
					MatcherEndpoint:    "",
				}},
			})
			require.NoError(t, err)

			if tt.setUpMocksAndEnv != nil {
				tt.setUpMocksAndEnv(t, tt.mocks)
			}
			pNode := nodes.NewPipeline(tt.mocks.clusterStore, tt.mocks.nodeDatastore, tt.enricher, tt.riskManager)
			pNodeInv := nodeinventory.NewPipeline(tt.mocks.clusterStore, tt.mocks.nodeDatastore, tt.enricher, tt.riskManager)
			pNodeIdx := newPipeline(tt.mocks.clusterStore, tt.mocks.nodeDatastore, tt.enricher, tt.riskManager)

			for i, op := range tt.operations {
				t.Logf("Running operation %d of %d", i+1, len(tt.operations))
				require.NoError(t, op(t, pNode, pNodeInv, pNodeIdx))
			}

			node, found, err := tt.mocks.nodeDatastore.GetNode(context.Background(), nodeID)
			assert.Equal(t, tt.wantNodeExists, found)
			require.NoError(t, err)
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

func createIndexReportWithKernel(kernelV string) *v4.IndexReport {
	return &v4.IndexReport{
		State:   "7", // IndexFinished
		Success: true,
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "1",
					Name:    kernelComponentName,
					Version: kernelV,
					Kind:    "binary",
					Source: &v4.Package{
						Name:    kernelComponentName,
						Version: kernelV,
						Kind:    "source",
						Source:  nil,
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
			},
			Repositories: []*v4.Repository{
				{
					Id:   "1",
					Name: "cpe:/o:redhat:enterprise_linux:9::fastdatapath",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*",
				},
			},
			Environments: map[string]*v4.Environment_List{"1": {Environments: []*v4.Environment{
				{
					PackageDb:     "sqlite:usr/share/rpm",
					IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					RepositoryIds: []string{"1"},
				},
			},
			}},
		},
	}
}

func createNodeIndexMsg(id, kernel string) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: id,
				Resource: &central.SensorEvent_IndexReport{
					IndexReport: createIndexReportWithKernel(kernel),
				},
			},
		},
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

func (f *fakeNodeScanner) GetNodeInventoryScan(*storage.Node, *storage.NodeInventory, *v4.IndexReport) (*storage.NodeScan, error) {
	f.requestedScan = true
	return nodeScanFixtureWithKernel("v2"), nil
}

func (*fakeNodeScanner) TestNodeScanner() error {
	return nil
}

func (*fakeNodeScanner) Type() string {
	return types.Clairify
}

func (*fakeNodeScanner) Name() string {
	return "name"
}

var _ types.NodeScanner = (*fakeNodeScannerv4)(nil)

type fakeNodeScannerv4 struct {
}

func (f fakeNodeScannerv4) MaxConcurrentNodeScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f fakeNodeScannerv4) Name() string {
	return "fake Scanner v4"
}

func (f fakeNodeScannerv4) GetNodeInventoryScan(_ *storage.Node, _ *storage.NodeInventory, _ *v4.IndexReport) (*storage.NodeScan, error) {
	return nodeScanFixtureWithKernel("v4"), nil
}

func (f fakeNodeScannerv4) GetNodeScan(_ *storage.Node) (*storage.NodeScan, error) {
	return nil, nil
}

func (f fakeNodeScannerv4) TestNodeScanner() error {
	return nil
}

func (f fakeNodeScannerv4) Type() string {
	return types.ScannerV4
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
