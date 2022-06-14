//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/stackrox/central/node/datastore/internal/search"
	"github.com/stackrox/stackrox/central/node/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/ranking"
	mockRisks "github.com/stackrox/stackrox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/cve"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/postgres/pgtest"
	"github.com/stackrox/stackrox/pkg/postgres/schema"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/postgres/mapping"
	"github.com/stackrox/stackrox/pkg/search/scoped"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestNodeDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(NodePostgresDataStoreTestSuite))
}

type NodePostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	db        *pgxpool.Pool
	gormDB    *gorm.DB
	datastore DataStore
	mockRisk  *mockRisks.MockDataStore

	envIsolator *envisolator.EnvIsolator
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

	suite.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(suite.T()))
	storage := postgres.CreateTableAndNewStore(suite.ctx, suite.T(), suite.db, suite.gormDB, false)
	indexer := postgres.NewIndexer(suite.db)
	searcher := search.NewV2(storage, indexer)
	suite.datastore = NewWithPostgres(storage, indexer, searcher, suite.mockRisk, ranking.NodeRanker(), ranking.NodeComponentRanker())
}

func (suite *NodePostgresDataStoreTestSuite) TearDownSuite() {
	suite.envIsolator.RestoreAll()
	suite.db.Close()
	pgtest.CloseGormDB(suite.T(), suite.gormDB)
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
	mapping.RegisterCategoryToTable(v1.SearchCategory_NODE_VULNERABILITIES, schema.NodeCvesSchema)
	ctx := sac.WithAllAccess(context.Background())
	suite.upsertTestNodes(ctx)

	// Search by CVE.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve1", ""),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 2)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve3", ""),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve4", ""),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve1", ""),
		Level: v1.SearchCategory_NODE_VULNERABILITIES,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    cve.ID("cve3", ""),
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
	mapping.RegisterCategoryToTable(v1.SearchCategory_NODE_COMPONENTS, schema.NodeComponentsSchema)
	ctx := sac.WithAllAccess(context.Background())
	suite.upsertTestNodes(ctx)

	// Search by Component.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp1", "ver1", ""),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err := suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 2)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp3", "ver1", ""),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Len(results, 1)
	suite.Equal("id2", results[0].ID)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp4", "ver1", ""),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	suite.deleteTestNodes(ctx)

	// Ensure search does not find anything.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp1", "ver1", ""),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)

	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    scancomponent.ComponentID("comp3", "ver1", ""),
		Level: v1.SearchCategory_NODE_COMPONENTS,
	})
	results, err = suite.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
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

func getTestNodeForPostgres(id, name string) *storage.Node {
	return &storage.Node{
		Id:        id,
		Name:      name,
		ClusterId: id,
		Scan: &storage.NodeScan{
			ScanTime: types.TimestampNow(),
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
