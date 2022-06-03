package datastore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	componentCVEEdgeDackBox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/rox/central/cve/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/globalindex"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	componentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	nodeIndex "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeDackBox "github.com/stackrox/rox/central/nodecomponentedge/dackbox"
	nodeComponentEdgeIndex "github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestNodeDataStore(t *testing.T) {
	suite.Run(t, new(NodeDataStoreTestSuite))
}

type NodeDataStoreTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	blevePath string
	indexQ    queue.WaitableQueue
	datastore DataStore

	mockRisk *mockRisks.MockDataStore
}

func (suite *NodeDataStoreTestSuite) SetupSuite() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())

	suite.indexQ = queue.NewWaitableQueue()

	dacky, err := dackbox.NewRocksDBDackBox(suite.db, suite.indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNow("failed to create dackbox", err.Error())
	}

	suite.blevePath = suite.T().TempDir()
	blevePath := filepath.Join(suite.blevePath, "scorch.bleve")
	bleveIndex, err := globalindex.InitializeIndices("main", blevePath, globalindex.EphemeralIndex, "")
	if err != nil {
		suite.FailNow("failed to create bleve index", err.Error())
	}

	reg := indexer.NewWrapperRegistry()
	indexer.NewLazy(suite.indexQ, reg, bleveIndex, dacky.AckIndexed).Start()
	reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})
	reg.RegisterWrapper(componentDackBox.Bucket, componentIndex.Wrapper{})
	reg.RegisterWrapper(componentCVEEdgeDackBox.Bucket, componentCVEEdgeIndex.Wrapper{})
	reg.RegisterWrapper(nodeDackBox.Bucket, nodeIndex.Wrapper{})
	reg.RegisterWrapper(nodeComponentEdgeDackBox.Bucket, nodeComponentEdgeIndex.Wrapper{})

	suite.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(suite.T()))

	suite.datastore = New(dacky, concurrency.NewKeyFence(), bleveIndex, suite.mockRisk, ranking.NodeRanker(), ranking.ComponentRanker())
}

func (suite *NodeDataStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *NodeDataStoreTestSuite) TestBasicOps() {
	node := getTestNode("id1", "name1")

	readCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Node),
		))
	suite.Error(suite.datastore.UpsertNode(readCtx, node), "permission denied")

	// No permission to write nodes.
	imgCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
		))
	suite.Error(suite.datastore.UpsertNode(imgCtx, node), "permission denied")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Node),
	))

	// Upsert node.
	suite.NoError(suite.datastore.UpsertNode(ctx, node))

	// Get node.
	storedNode, exists, err := suite.datastore.GetNode(ctx, node.Id)
	suite.True(exists)
	suite.NoError(err)
	suite.NotNil(storedNode)
	for _, component := range node.GetScan().GetComponents() {
		for _, cve := range component.GetVulns() {
			cve.FirstSystemOccurrence = storedNode.GetLastUpdated()
			cve.VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
			cve.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_NODE_VULNERABILITY}
		}
	}
	suite.Equal(node, storedNode)

	// Exists tests.
	exists, err = suite.datastore.Exists(ctx, "id1")
	suite.NoError(err)
	suite.True(exists)
	exists, err = suite.datastore.Exists(ctx, "id2")
	suite.NoError(err)
	suite.False(exists)

	// Upsert old scan should not change data (save for node.LastUpdated).
	olderNode := node.Clone()
	olderNode.GetScan().GetScanTime().Seconds = olderNode.GetScan().GetScanTime().GetSeconds() - 500
	olderNode.Scan = &storage.NodeScan{}
	suite.NoError(suite.datastore.UpsertNode(ctx, olderNode))
	storedNode, exists, err = suite.datastore.GetNode(ctx, olderNode.Id)
	suite.True(exists)
	suite.NoError(err)
	// Node is updated.
	node.LastUpdated = storedNode.GetLastUpdated()
	// Scan data is unchanged.
	suite.Equal(node, storedNode)

	newNode := node.Clone()
	newNode.Id = "id2"

	// Upsert new node.
	suite.NoError(suite.datastore.UpsertNode(ctx, newNode))

	// Exists test.
	exists, err = suite.datastore.Exists(ctx, "id2")
	suite.NoError(err)
	suite.True(exists)

	// Get new node.
	storedNode, exists, err = suite.datastore.GetNode(ctx, newNode.Id)
	suite.True(exists)
	suite.NoError(err)
	suite.NotNil(storedNode)
	suite.Equal(newNode, storedNode)

	// Count nodes.
	count, err := suite.datastore.CountNodes(ctx)
	suite.NoError(err)
	suite.Equal(2, count)

	// Get batch.
	nodes, err := suite.datastore.GetNodesBatch(ctx, []string{"id1", "id2"})
	suite.NoError(err)
	suite.Len(nodes, 2)
	suite.ElementsMatch([]*storage.Node{node, newNode}, nodes)

	// Delete both nodes.
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id1", storage.RiskSubjectType_NODE).Return(nil)
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id2", storage.RiskSubjectType_NODE).Return(nil)
	suite.NoError(suite.datastore.DeleteNodes(ctx, "id1", "id2"))

	// Exists tests.
	exists, err = suite.datastore.Exists(ctx, "id1")
	suite.NoError(err)
	suite.False(exists)
	exists, err = suite.datastore.Exists(ctx, "id2")
	suite.NoError(err)
	suite.False(exists)
}

func (suite *NodeDataStoreTestSuite) TestBasicSearch() {
	node := getTestNode("id1", "name1")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Node),
	))

	// Basic unscoped search.
	results, err := suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	// Upsert node.
	suite.NoError(suite.datastore.UpsertNode(ctx, node))

	// Ensure the CVEs are indexed.
	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Basic unscoped search.
	results, err = suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    node.GetId(),
		Level: v1.SearchCategory_NODES,
	})

	// Basic scoped search.
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	// Search Nodes.
	nodes, err := suite.datastore.SearchRawNodes(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.NotNil(nodes)
	suite.Len(nodes, 1)
	for _, component := range node.GetScan().GetComponents() {
		for _, cve := range component.GetVulns() {
			cve.FirstSystemOccurrence = nodes[0].GetLastUpdated()
			cve.VulnerabilityType = storage.EmbeddedVulnerability_UNKNOWN_VULNERABILITY
			cve.VulnerabilityTypes = []storage.EmbeddedVulnerability_VulnerabilityType{storage.EmbeddedVulnerability_NODE_VULNERABILITY}
		}
	}
	suite.Equal(node, nodes[0])

	// Upsert new node.
	newNode := getTestNode("id2", "name2")
	newNode.GetScan().Components = append(newNode.GetScan().GetComponents(), &storage.EmbeddedNodeScanComponent{
		Name:    "comp3",
		Version: "ver1",
		Vulns: []*storage.EmbeddedVulnerability{
			{
				Cve:               "cve3",
				VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
			},
		},
	})
	suite.NoError(suite.datastore.UpsertNode(ctx, newNode))

	// Ensure the CVEs are indexed.
	indexingDone = concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()

	// Search multiple nodes.
	nodes, err = suite.datastore.SearchRawNodes(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(nodes, 2)

	// Search for just one node.
	nodes, err = suite.datastore.SearchRawNodes(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(nodes, 1)
	suite.Equal(node, nodes[0])

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	results, err = suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *NodeDataStoreTestSuite) TestSearchByVuln() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Node),
	))
	suite.upsertTestNodes(ctx)

	// Search by CVE.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    "cve1",
		Level: v1.SearchCategory_VULNERABILITIES,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 2)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve3",
		Level: v1.SearchCategory_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve4",
		Level: v1.SearchCategory_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve1",
		Level: v1.SearchCategory_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve3",
		Level: v1.SearchCategory_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	results, err = suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *NodeDataStoreTestSuite) TestSearchByNodeCVEEdge() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Node),
	))
	suite.upsertTestNodes(ctx)

	// Search by NodeCVEEdge.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    edges.EdgeID{ParentID: "id1", ChildID: "cve1"}.ToString(),
		Level: v1.SearchCategory_NODE_VULN_EDGE,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id1", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    edges.EdgeID{ParentID: "id1", ChildID: "cve2"}.ToString(),
		Level: v1.SearchCategory_NODE_VULN_EDGE,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id1", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    edges.EdgeID{ParentID: "id2", ChildID: "cve3"}.ToString(),
		Level: v1.SearchCategory_NODE_VULN_EDGE,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	results, err = suite.datastore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *NodeDataStoreTestSuite) TestSearchByComponent() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Node),
	))
	suite.upsertTestNodes(ctx)

	// Search by Component.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp1", "ver1", ""),
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 2)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp3", "ver1", ""),
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp4", "ver1", ""),
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp1", "ver1", ""),
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp3", "ver1", ""),
		Level: v1.SearchCategory_IMAGE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *NodeDataStoreTestSuite) SearchByCluster() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Node),
	))
	suite.upsertTestNodes(ctx)

	// Search by Cluster.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    "id1",
		Level: v1.SearchCategory_CLUSTERS,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id1", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "id2",
		Level: v1.SearchCategory_CLUSTERS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "id3",
		Level: v1.SearchCategory_CLUSTERS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "id1",
		Level: v1.SearchCategory_CLUSTERS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "id2",
		Level: v1.SearchCategory_CLUSTERS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func getTestNode(id, name string) *storage.Node {
	return &storage.Node{
		Id:        id,
		Name:      name,
		ClusterId: id,
		Scan: &storage.NodeScan{
			ScanTime: types.TimestampNow(),
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Name:    "comp1",
					Version: "ver1",
					Vulns:   []*storage.EmbeddedVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver3",
							},
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve:               "cve1",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "ver2",
							},
						},
						{
							Cve:               "cve2",
							VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
						},
					},
				},
			},
		},
		RiskScore: 30,
		Priority:  1,
	}
}

func (suite *NodeDataStoreTestSuite) upsertTestNodes(ctx context.Context) {
	node := getTestNode("id1", "name1")

	// Upsert node.
	suite.NoError(suite.datastore.UpsertNode(ctx, node))

	// Upsert new node.
	newNode := getTestNode("id2", "name2")
	newNode.GetScan().Components = append(newNode.GetScan().GetComponents(), &storage.EmbeddedNodeScanComponent{
		Name:    "comp3",
		Version: "ver1",
		Vulns: []*storage.EmbeddedVulnerability{
			{
				Cve:               "cve3",
				VulnerabilityType: storage.EmbeddedVulnerability_NODE_VULNERABILITY,
			},
		},
	})
	suite.NoError(suite.datastore.UpsertNode(ctx, newNode))

	// Ensure the CVEs are indexed.
	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()
}

func (suite *NodeDataStoreTestSuite) deleteTestNodes(ctx context.Context) {
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id1", storage.RiskSubjectType_NODE).Return(nil)
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id2", storage.RiskSubjectType_NODE).Return(nil)
	suite.NoError(suite.datastore.DeleteNodes(ctx, "id1", "id2"))

	indexingDone := concurrency.NewSignal()
	suite.indexQ.PushSignal(&indexingDone)
	indexingDone.Wait()
}
