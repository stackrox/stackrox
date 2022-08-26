//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"sort"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	nodeCVEDS "github.com/stackrox/rox/central/cve/node/datastore"
	nodeCVESearch "github.com/stackrox/rox/central/cve/node/datastore/search"
	nodeCVEPostgres "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	"github.com/stackrox/rox/central/node/datastore/search"
	"github.com/stackrox/rox/central/node/datastore/store/postgres"
	nodeComponentDS "github.com/stackrox/rox/central/nodecomponent/datastore"
	nodeComponentSearch "github.com/stackrox/rox/central/nodecomponent/datastore/search"
	nodeComponentPostgres "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	pkgCVE "github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/nodes/converter"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestNodeDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(NodePostgresDataStoreTestSuite))
}

type NodePostgresDataStoreTestSuite struct {
	suite.Suite

	ctx                context.Context
	db                 *pgxpool.Pool
	gormDB             *gorm.DB
	datastore          DataStore
	mockCtrl           *gomock.Controller
	mockRisk           *mockRisks.MockDataStore
	componentDataStore nodeComponentDS.DataStore
	nodeCVEDataStore   nodeCVEDS.DataStore
	envIsolator        *envisolator.EnvIsolator
}

func (suite *NodePostgresDataStoreTestSuite) SetupSuite() {
	suite.envIsolator = envisolator.NewEnvIsolator(suite.T())
	suite.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		suite.T().Skip("Skip postgres tests")
		suite.T().SkipNow()
	}

	suite.ctx = context.Background()

	source := pgtest.GetConnectionString(suite.T())
	config, err := pgxpool.ParseConfig(source)
	suite.Require().NoError(err)

	pool, err := pgxpool.ConnectConfig(suite.ctx, config)
	suite.NoError(err)
	suite.gormDB = pgtest.OpenGormDB(suite.T(), source)
	suite.db = pool
}

func (suite *NodePostgresDataStoreTestSuite) SetupTest() {
	postgres.Destroy(suite.ctx, suite.db)

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockRisk = mockRisks.NewMockDataStore(suite.mockCtrl)
	storage := postgres.CreateTableAndNewStore(suite.ctx, suite.T(), suite.db, suite.gormDB, false)
	indexer := postgres.NewIndexer(suite.db)
	searcher := search.NewV2(storage, indexer)
	suite.datastore = NewWithPostgres(storage, indexer, searcher, suite.mockRisk, ranking.NewRanker(), ranking.NewRanker())

	componentStorage := nodeComponentPostgres.CreateTableAndNewStore(suite.ctx, suite.db, suite.gormDB)
	componentIndexer := nodeComponentPostgres.NewIndexer(suite.db)
	componentSearcher := nodeComponentSearch.New(componentStorage, componentIndexer)
	suite.componentDataStore = nodeComponentDS.New(componentStorage, componentIndexer, componentSearcher, suite.mockRisk, ranking.NewRanker())

	cveStorage := nodeCVEPostgres.CreateTableAndNewStore(suite.ctx, suite.db, suite.gormDB)
	cveIndexer := nodeCVEPostgres.NewIndexer(suite.db)
	cveSearcher := nodeCVESearch.New(cveStorage, cveIndexer)
	cveDataStore, err := nodeCVEDS.New(cveStorage, cveIndexer, cveSearcher, concurrency.NewKeyFence())
	suite.NoError(err)
	suite.nodeCVEDataStore = cveDataStore
}

func (suite *NodePostgresDataStoreTestSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
	suite.db.Close()
	pgtest.CloseGormDB(suite.T(), suite.gormDB)
	suite.envIsolator.RestoreAll()
}

func (suite *NodePostgresDataStoreTestSuite) TestBasicOps() {
	node := getTestNodeForPostgres("id1", "name1")
	allowAllCtx := sac.WithAllAccess(context.Background())

	// Upsert.
	suite.NoError(suite.datastore.UpsertNode(allowAllCtx, node))

	// Get node.
	storedNode, exists, err := suite.datastore.GetNode(allowAllCtx, node.Id)
	suite.True(exists)
	suite.NoError(err)
	suite.NotNil(storedNode)
	for _, component := range node.GetScan().GetComponents() {
		for _, cve := range component.GetVulnerabilities() {
			cve.CveBaseInfo.CreatedAt = storedNode.GetLastUpdated()
		}
	}
	suite.EqualValues(node, storedNode)

	// Exists tests.
	exists, err = suite.datastore.Exists(allowAllCtx, "id1")
	suite.NoError(err)
	suite.True(exists)
	exists, err = suite.datastore.Exists(allowAllCtx, "id2")
	suite.NoError(err)
	suite.False(exists)

	// Upsert old scan should not change data (save for node.LastUpdated).
	olderNode := node.Clone()
	olderNode.GetScan().GetScanTime().Seconds = olderNode.GetScan().GetScanTime().GetSeconds() - 500
	suite.NoError(suite.datastore.UpsertNode(allowAllCtx, olderNode))
	storedNode, exists, err = suite.datastore.GetNode(allowAllCtx, olderNode.Id)
	suite.True(exists)
	suite.NoError(err)
	// Node is updated.
	node.LastUpdated = storedNode.GetLastUpdated()
	// Scan data is unchanged.
	suite.Equal(node, storedNode)

	newNode := node.Clone()
	newNode.Id = "id2"

	// Upsert new node.
	suite.NoError(suite.datastore.UpsertNode(allowAllCtx, newNode))

	// Exists test.
	exists, err = suite.datastore.Exists(allowAllCtx, "id2")
	suite.NoError(err)
	suite.True(exists)

	// Get new node.
	storedNode, exists, err = suite.datastore.GetNode(allowAllCtx, newNode.Id)
	suite.True(exists)
	suite.NoError(err)
	suite.NotNil(storedNode)
	suite.Equal(newNode, storedNode)

	// Count nodes.
	count, err := suite.datastore.CountNodes(allowAllCtx)
	suite.NoError(err)
	suite.Equal(2, count)

	// Get batch.
	nodes, err := suite.datastore.GetNodesBatch(allowAllCtx, []string{"id1", "id2"})
	suite.NoError(err)
	suite.Len(nodes, 2)
	suite.ElementsMatch([]*storage.Node{node, newNode}, nodes)

	// Delete both nodes.
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id1", storage.RiskSubjectType_NODE).Return(nil)
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id2", storage.RiskSubjectType_NODE).Return(nil)
	suite.NoError(suite.datastore.DeleteNodes(allowAllCtx, "id1", "id2"))

	// Exists tests.
	exists, err = suite.datastore.Exists(allowAllCtx, "id1")
	suite.NoError(err)
	suite.False(exists)
	exists, err = suite.datastore.Exists(allowAllCtx, "id2")
	suite.NoError(err)
	suite.False(exists)
}

func (suite *NodePostgresDataStoreTestSuite) TestBasicSearch() {
	node := getTestNodeForPostgres("id1", "name1")
	allowAllCtx := sac.WithAllAccess(context.Background())

	// Basic unscoped search.
	results, err := suite.datastore.Search(allowAllCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	// Upsert node.
	suite.NoError(suite.datastore.UpsertNode(allowAllCtx, node))

	// Basic unscoped search.
	results, err = suite.datastore.Search(allowAllCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)

	scopedCtx := scoped.Context(allowAllCtx, scoped.Scope{
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
		for _, cve := range component.GetVulnerabilities() {
			cve.CveBaseInfo.CreatedAt = nodes[0].GetLastUpdated()
		}
	}
	suite.Equal(node, nodes[0])

	// Upsert new node.
	newNode := getTestNodeForPostgres("id2", "name2")
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
	suite.NoError(suite.datastore.UpsertNode(allowAllCtx, newNode))

	// Search multiple nodes.
	nodes, err = suite.datastore.SearchRawNodes(allowAllCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(nodes, 2)

	// Search for just one node.
	nodes, err = suite.datastore.SearchRawNodes(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(nodes, 1)
	suite.Equal(node, nodes[0])

	suite.deleteTestNodes(allowAllCtx)

	// Ensure search does not find anything.
	results, err = suite.datastore.Search(allowAllCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

func (suite *NodePostgresDataStoreTestSuite) TestSearchByVuln() {
	ctx := sac.WithAllAccess(context.Background())
	suite.upsertTestNodes(ctx)

	// Search by CVE.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve1", "ubuntu"),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 2)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve3", "ubuntu"),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve4", "ubuntu"),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve1", "ubuntu"),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve3", "ubuntu"),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
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

func (suite *NodePostgresDataStoreTestSuite) TestSearchByComponent() {
	ctx := sac.WithAllAccess(context.Background())
	suite.upsertTestNodes(ctx)

	// Search by Component.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp1", "ver1", "ubuntu"),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 2)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp3", "ver1", "ubuntu"),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp4", "ver1", "ubuntu"),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp1", "ver1", "ubuntu"),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp3", "ver1", "ubuntu"),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}

// Test sort by Component search label sorts by Component+Version to ensure backward compatibility.
func (suite *NodePostgresDataStoreTestSuite) TestSortByComponent() {
	ctx := sac.WithAllAccess(context.Background())
	node := fixtures.GetNodeWithUniqueComponents()
	converter.FillV2NodeVulnerabilities(node)
	componentIDs := make([]string, 0, len(node.GetScan().GetComponents()))
	for _, component := range node.GetScan().GetComponents() {
		componentIDs = append(componentIDs,
			scancomponent.ComponentID(
				component.GetName(),
				component.GetVersion(),
				node.GetScan().GetOperatingSystem(),
			))
	}

	suite.NoError(suite.datastore.UpsertNode(ctx, node))

	// Verify sort by Component search label is transformed to sort by Component+Version.
	query := pkgSearch.EmptyQuery()
	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: pkgSearch.Component.String(),
			},
		},
	}
	// Component ID is Name+Version+Operating System. Therefore, sort by ID is same as Component+Version.
	sort.SliceStable(componentIDs, func(i, j int) bool {
		return componentIDs[i] < componentIDs[j]
	})
	results, err := suite.componentDataStore.Search(ctx, query)
	suite.NoError(err)
	suite.Equal(componentIDs, pkgSearch.ResultsToIDs(results))

	// Verify reverse sort.
	sort.SliceStable(componentIDs, func(i, j int) bool {
		return componentIDs[i] > componentIDs[j]
	})
	query.Pagination.SortOptions[0].Reversed = true
	results, err = suite.componentDataStore.Search(ctx, query)
	suite.NoError(err)
	suite.Equal(componentIDs, pkgSearch.ResultsToIDs(results))

	// Verify sorting by fields of different table works correctly.
	results, err = suite.datastore.Search(ctx, query)
	suite.NoError(err)
	suite.Equal(1, len(results))

	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: pkgSearch.CVE.String(),
			},
		},
	}
	results, err = suite.componentDataStore.Search(ctx, query)
	suite.NoError(err)
	suite.Equal(len(componentIDs), len(results))
}

func (suite *NodePostgresDataStoreTestSuite) upsertTestNodes(ctx context.Context) {
	node := getTestNodeForPostgres("id1", "name1")

	// Upsert node.
	suite.NoError(suite.datastore.UpsertNode(ctx, node))

	// Upsert new node.
	newNode := getTestNodeForPostgres("id2", "name2")
	newNode.GetScan().Components = append(newNode.GetScan().GetComponents(), &storage.EmbeddedNodeScanComponent{
		Name:    "comp3",
		Version: "ver1",
		Vulnerabilities: []*storage.NodeVulnerability{
			{
				CveBaseInfo: &storage.CVEInfo{
					Cve: "cve3",
				},
			},
		},
	})
	suite.NoError(suite.datastore.UpsertNode(ctx, newNode))
}

func (suite *NodePostgresDataStoreTestSuite) deleteTestNodes(ctx context.Context) {
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id1", storage.RiskSubjectType_NODE).Return(nil)
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), "id2", storage.RiskSubjectType_NODE).Return(nil)
	suite.NoError(suite.datastore.DeleteNodes(ctx, "id1", "id2"))
}

func (suite *NodePostgresDataStoreTestSuite) TestOrphanedNodeTreeDeletion() {
	ctx := sac.WithAllAccess(context.Background())
	testNode := fixtures.GetNodeWithUniqueComponents()
	converter.MoveNodeVulnsToNewField(testNode)
	suite.NoError(suite.datastore.UpsertNode(ctx, testNode))

	storedNode, found, err := suite.datastore.GetNode(ctx, testNode.GetId())
	suite.NoError(err)
	suite.True(found)
	for _, component := range testNode.GetScan().GetComponents() {
		for _, cve := range component.GetVulnerabilities() {
			cve.CveBaseInfo.CreatedAt = storedNode.GetLastUpdated()
		}
	}
	testNode.Priority = 1
	suite.Equal(testNode, storedNode)

	// Verify that new scan with less components cleans up the old relations correctly.
	testNode.Scan.ScanTime = types.TimestampNow()
	testNode.Scan.Components = testNode.Scan.Components[:len(testNode.Scan.Components)-1]
	cveIDsSet := set.NewStringSet()
	for _, component := range testNode.GetScan().GetComponents() {
		for _, cve := range component.GetVulnerabilities() {
			cveIDsSet.Add(pkgCVE.ID(cve.GetCveBaseInfo().GetCve(), testNode.GetScan().GetOperatingSystem()))
		}
	}
	suite.NoError(suite.datastore.UpsertNode(ctx, testNode))

	// Verify node is built correctly.
	storedNode, found, err = suite.datastore.GetNode(ctx, testNode.GetId())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(testNode, storedNode)

	// Verify orphaned node components are removed.
	count, err := suite.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Equal(len(testNode.Scan.Components), count)

	// Verify orphaned node vulnerabilities are removed.
	results, err := suite.nodeCVEDataStore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.ElementsMatch(cveIDsSet.AsSlice(), pkgSearch.ResultsToIDs(results))

	testNode2 := testNode.Clone()
	testNode2.Id = "2"
	suite.NoError(suite.datastore.UpsertNode(ctx, testNode2))
	storedNode, found, err = suite.datastore.GetNode(ctx, testNode2.GetId())
	suite.NoError(err)
	suite.True(found)
	testNode2.Priority = 1
	suite.Equal(testNode2, storedNode)

	// Verify that number of node components remains unchanged since both nodes have same components.
	count, err = suite.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Equal(len(testNode.Scan.Components), count)

	// Verify that number of node vulnerabilities remains unchanged since both nodes have same vulns.
	results, err = suite.nodeCVEDataStore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.ElementsMatch(cveIDsSet.AsSlice(), pkgSearch.ResultsToIDs(results))

	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), testNode.GetId(), gomock.Any()).Return(nil)
	suite.NoError(suite.datastore.DeleteNodes(ctx, testNode.GetId()))

	// Verify that second node is still constructed correctly.
	storedNode, found, err = suite.datastore.GetNode(ctx, testNode2.GetId())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(testNode2, storedNode)

	// Set all components to contain same cve.
	for _, component := range testNode2.GetScan().GetComponents() {
		component.Vulnerabilities = []*storage.NodeVulnerability{
			{CveBaseInfo: &storage.CVEInfo{Cve: "cve"}},
		}
	}
	testNode2.Scan.ScanTime = types.TimestampNow()

	suite.NoError(suite.datastore.UpsertNode(ctx, testNode2))
	storedNode, found, err = suite.datastore.GetNode(ctx, testNode2.GetId())
	suite.NoError(err)
	suite.True(found)
	for _, component := range testNode2.GetScan().GetComponents() {
		// Components and Vulns are deduped, therefore, update testNode structure.
		for _, cve := range component.GetVulnerabilities() {
			cve.CveBaseInfo.CreatedAt = storedNode.GetLastUpdated()
		}
	}
	suite.Equal(testNode2, storedNode)

	// Verify orphaned node components are removed.
	count, err = suite.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Equal(len(testNode2.Scan.Components), count)

	// Verify orphaned node vulnerabilities are removed.
	results, err = suite.nodeCVEDataStore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.ElementsMatch([]string{pkgCVE.ID("cve", "")}, pkgSearch.ResultsToIDs(results))

	// Verify that new scan with less components cleans up the old relations correctly.
	testNode2.Scan.ScanTime = types.TimestampNow()
	testNode2.Scan.Components = testNode2.Scan.Components[:len(testNode2.Scan.Components)-1]
	suite.NoError(suite.datastore.UpsertNode(ctx, testNode2))

	// Verify node is built correctly.
	storedNode, found, err = suite.datastore.GetNode(ctx, testNode2.GetId())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(testNode2, storedNode)

	// Verify orphaned node components are removed.
	count, err = suite.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Equal(len(testNode2.Scan.Components), count)

	// Verify no vulnerability is removed since all vulns are still connected.
	results, err = suite.nodeCVEDataStore.Search(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.ElementsMatch([]string{pkgCVE.ID("cve", "")}, pkgSearch.ResultsToIDs(results))

	// Verify that new scan with no components and vulns cleans up the old relations correctly.
	testNode2.Scan.ScanTime = types.TimestampNow()
	testNode2.Scan.Components = nil
	suite.NoError(suite.datastore.UpsertNode(ctx, testNode2))

	// Verify node is built correctly.
	storedNode, found, err = suite.datastore.GetNode(ctx, testNode2.GetId())
	suite.NoError(err)
	suite.True(found)
	suite.Equal(testNode2, storedNode)

	// Verify no components exist.
	count, err = suite.componentDataStore.Count(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Equal(0, count)

	// Verify no vulnerabilities exist.
	count, err = suite.nodeCVEDataStore.Count(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Equal(0, count)

	// Delete node.
	suite.mockRisk.EXPECT().RemoveRisk(gomock.Any(), testNode2.GetId(), gomock.Any()).Return(nil)
	suite.NoError(suite.datastore.DeleteNodes(ctx, testNode2.GetId()))

	// Verify no node exist.
	count, err = suite.datastore.Count(ctx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Equal(0, count)
}

func getTestNodeForPostgres(id, name string) *storage.Node {
	return &storage.Node{
		Id:        id,
		Name:      name,
		ClusterId: id,
		Scan: &storage.NodeScan{
			ScanTime:        types.TimestampNow(),
			OperatingSystem: "ubuntu",
			Components: []*storage.EmbeddedNodeScanComponent{
				{
					Name:            "comp1",
					Version:         "ver1",
					Vulnerabilities: []*storage.NodeVulnerability{},
				},
				{
					Name:    "comp1",
					Version: "ver2",
					Vulnerabilities: []*storage.NodeVulnerability{
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve1",
							},
						},
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve2",
							},
							SetFixedBy: &storage.NodeVulnerability_FixedBy{
								FixedBy: "ver3",
							},
						},
					},
				},
				{
					Name:    "comp2",
					Version: "ver1",
					Vulnerabilities: []*storage.NodeVulnerability{
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve1",
							},
							SetFixedBy: &storage.NodeVulnerability_FixedBy{
								FixedBy: "ver2",
							},
						},
						{
							CveBaseInfo: &storage.CVEInfo{
								Cve: "cve2",
							},
						},
					},
				},
			},
		},
		RiskScore: 30,
		Priority:  1,
	}
}
