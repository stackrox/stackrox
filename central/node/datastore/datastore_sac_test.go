//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestNodeDatastoreSAC(t *testing.T) {
	suite.Run(t, new(nodeDatastoreSACSuite))
}

type nodeDatastoreSACSuite struct {
	suite.Suite

	datastore  DataStore
	optionsMap searchPkg.OptionsMap

	// Elements for postgres mode
	pgtestbase *pgtest.TestPostgres

	testContexts map[string]context.Context

	testNodeIDs map[string][]string
	testNodes   map[string]*storage.Node
}

func (s *nodeDatastoreSACSuite) SetupSuite() {
	var err error
	s.pgtestbase = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgtestbase)
	s.datastore = GetTestPostgresDataStore(s.T(), s.pgtestbase.DB)
	s.Require().NoError(err)
	s.optionsMap = schema.NodesSchema.OptionsMap

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
}

func (s *nodeDatastoreSACSuite) TearDownSuite() {
	s.pgtestbase.DB.Close()
}

func (s *nodeDatastoreSACSuite) SetupTest() {
	s.testNodeIDs = make(map[string][]string, 0)
	s.testNodes = make(map[string]*storage.Node, 0)

	s.initTestResourceSet()
}

func (s *nodeDatastoreSACSuite) TearDownTest() {
	for _, nodeIds := range s.testNodeIDs {
		s.Require().NoError(s.datastore.DeleteNodes(s.testContexts[testutils.UnrestrictedReadWriteCtx], nodeIds...))
	}
}

func (s *nodeDatastoreSACSuite) addTestNode(clusterID string) string {
	nodeID := uuid.NewV4().String()
	node := fixtures.GetScopedNode(nodeID, clusterID)
	node.Priority = 1

	errUpsert := s.datastore.UpsertNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], node)
	s.Require().NoError(errUpsert)
	s.testNodeIDs[clusterID] = append(s.testNodeIDs[clusterID], nodeID)
	s.testNodes[nodeID] = node

	return nodeID
}

func (s *nodeDatastoreSACSuite) initTestResourceSet() {
	clusters := []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3}

	const numberOfNodes = 3
	for _, clusterID := range clusters {
		s.testNodeIDs[clusterID] = make([]string, 0, numberOfNodes)
		for i := 0; i < numberOfNodes; i++ {
			s.addTestNode(clusterID)
		}
	}
}

type sacMultiNodeTest struct {
	Context            context.Context
	ValidClusterScope  bool
	ExpectedClusterIds []string
}

func getSACMultiNodeTestCases(baseContext context.Context, _ *testing.T, validClusterIDs []string, wrongClusterID string, resources ...permissions.ResourceMetadata) map[string]sacMultiNodeTest {
	resourceHandles := make([]permissions.ResourceHandle, 0, len(resources))
	for _, r := range resources {
		resourceHandles = append(resourceHandles, r)
	}

	return map[string]sacMultiNodeTest{
		"(full) read-only can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...))),
			ValidClusterScope:  false,
			ExpectedClusterIds: []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3},
		},
		"full read-write can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...))),
			ValidClusterScope:  false,
			ExpectedClusterIds: []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3},
		},
		"full read-write on wrong cluster cannot get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(wrongClusterID))),
			ValidClusterScope:  false,
			ExpectedClusterIds: []string{},
		},
		"read-write on wrong cluster and partial namespace access cannot get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(wrongClusterID),
					sac.NamespaceScopeKeys("someNamespace"))),
			ValidClusterScope:  false,
			ExpectedClusterIds: []string{},
		},
		"read-only on right cluster can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterIDs...))),
			ValidClusterScope:  true,
			ExpectedClusterIds: validClusterIDs,
		},
		"full read-write on right cluster can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterIDs...))),
			ValidClusterScope:  true,
			ExpectedClusterIds: validClusterIDs,
		},
		"read-write on the right cluster and partial namespace access can get": {
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resourceHandles...),
					sac.ClusterScopeKeys(validClusterIDs...),
					sac.NamespaceScopeKeys("someNamespace"))),
			ValidClusterScope:  true,
			ExpectedClusterIds: validClusterIDs,
		},
	}
}

func (s *nodeDatastoreSACSuite) TestExists() {
	clusterID := testconsts.Cluster2
	nodeID := s.testNodeIDs[clusterID][2]

	cases := testutils.GenericClusterSACGetTestCases(context.Background(), s.T(), clusterID, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			exists, err := s.datastore.Exists(ctx, nodeID)
			s.NoError(err)
			s.Equal(c.ExpectedFound, exists)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestGetNode() {
	clusterID := testconsts.Cluster2
	nodeID := s.testNodeIDs[clusterID][1]

	cases := testutils.GenericClusterSACGetTestCases(context.Background(), s.T(), clusterID, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			fetchedNode, found, err := s.datastore.GetNode(ctx, nodeID)
			s.NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Require().NotNil(fetchedNode)

				// Priority can have updated value, and we want to ignore it.
				fetchedNode.Priority = s.testNodes[nodeID].Priority
				s.Equal(*s.testNodes[nodeID], *fetchedNode)
			} else {
				s.False(found)
				s.Nil(fetchedNode)
			}
		})
	}
}

func (s *nodeDatastoreSACSuite) TestCountNodes() {
	clusterIDs := []string{testconsts.Cluster1, testconsts.Cluster3}

	s.addTestNode(clusterIDs[0])

	cases := getSACMultiNodeTestCases(context.Background(), s.T(), clusterIDs, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			numOfNodes, err := s.datastore.CountNodes(ctx)
			s.NoError(err)

			// No accessible clusters.
			if len(c.ExpectedClusterIds) == 0 {
				s.Equal(0, numOfNodes)

				return
			}

			// Can access target clusters.
			if c.ValidClusterScope {
				total := 0
				for _, clusterID := range clusterIDs {
					total += len(s.testNodeIDs[clusterID])
				}
				s.Equal(total, numOfNodes)

				return
			}

			// Can access to all clusters.
			s.Equal(len(s.testNodes), numOfNodes)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestCount() {
	clusterIDs := []string{testconsts.Cluster1, testconsts.Cluster3}

	s.addTestNode(clusterIDs[0])

	cases := getSACMultiNodeTestCases(context.Background(), s.T(), clusterIDs, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			numOfNodes, err := s.datastore.Count(ctx, searchPkg.EmptyQuery())
			s.NoError(err)

			// No accessible clusters.
			if len(c.ExpectedClusterIds) == 0 {
				s.Equal(0, numOfNodes)

				return
			}

			// Can access target clusters.
			if c.ValidClusterScope {
				total := 0
				for _, clusterID := range clusterIDs {
					total += len(s.testNodeIDs[clusterID])
				}
				s.Equal(total, numOfNodes)

				return
			}

			// Can access to all clusters.
			s.Equal(len(s.testNodes), numOfNodes)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestSearch() {
	clusterIDs := []string{testconsts.Cluster1, testconsts.Cluster3}

	cases := getSACMultiNodeTestCases(context.Background(), s.T(), clusterIDs, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			results, err := s.datastore.Search(ctx, nil)
			s.NoError(err)

			fetchedNodeIDs := make([]string, 0, len(results))
			for _, result := range results {
				fetchedNodeIDs = append(fetchedNodeIDs, result.ID)
			}

			var expectedNodeIds []string
			for _, expectedClusterID := range c.ExpectedClusterIds {
				expectedNodeIds = append(expectedNodeIds, s.testNodeIDs[expectedClusterID]...)
			}

			s.ElementsMatch(expectedNodeIds, fetchedNodeIDs)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestSearchNodes() {
	clusterIDs := []string{testconsts.Cluster1, testconsts.Cluster3}

	cases := getSACMultiNodeTestCases(context.Background(), s.T(), clusterIDs, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			searchResults, err := s.datastore.SearchNodes(ctx, nil)
			s.NoError(err)

			fetchedNodeIDs := make([]string, 0, len(searchResults))
			for _, result := range searchResults {
				fetchedNodeIDs = append(fetchedNodeIDs, result.GetId())
			}

			var expectedNodeIds []string
			for _, expectedClusterID := range c.ExpectedClusterIds {
				expectedNodeIds = append(expectedNodeIds, s.testNodeIDs[expectedClusterID]...)
			}

			s.ElementsMatch(expectedNodeIds, fetchedNodeIDs)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestSearchRawNodes() {
	clusterIDs := []string{testconsts.Cluster1, testconsts.Cluster3}

	cases := getSACMultiNodeTestCases(context.Background(), s.T(), clusterIDs, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			nodes, err := s.datastore.SearchRawNodes(ctx, nil)
			s.NoError(err)

			fetchedNodeIDs := make([]string, 0, len(nodes))
			for _, node := range nodes {
				fetchedNodeIDs = append(fetchedNodeIDs, node.GetId())

				s.Equal(s.testNodes[node.GetId()].GetClusterId(), node.GetClusterId())
			}

			var expectedNodeIds []string
			for _, expectedClusterID := range c.ExpectedClusterIds {
				expectedNodeIds = append(expectedNodeIds, s.testNodeIDs[expectedClusterID]...)
			}

			s.ElementsMatch(expectedNodeIds, fetchedNodeIDs)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestGetNodesBatch() {
	clusterIDs := []string{testconsts.Cluster1, testconsts.Cluster3}

	batchNodeIds := make([]string, 0, len(s.testNodes))
	for _, nodeIDs := range s.testNodeIDs {
		oddNode := true
		for _, nodeID := range nodeIDs {
			if oddNode {
				batchNodeIds = append(batchNodeIds, nodeID)
			}
			oddNode = !oddNode
		}
	}

	cases := getSACMultiNodeTestCases(context.Background(), s.T(), clusterIDs, testconsts.WrongCluster, resources.Node)
	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			nodes, err := s.datastore.GetNodesBatch(ctx, batchNodeIds)
			s.NoError(err)

			fetchedNodeIDs := make([]string, 0, len(nodes))
			for _, node := range nodes {
				fetchedNodeIDs = append(fetchedNodeIDs, node.GetId())

				s.Equal(s.testNodes[node.GetId()].GetClusterId(), node.GetClusterId())
			}

			var expectedNodeIds []string
			for _, expectedClusterID := range c.ExpectedClusterIds {
				oddNode := true
				for _, expectedNodeID := range s.testNodeIDs[expectedClusterID] {
					if oddNode {
						expectedNodeIds = append(expectedNodeIds, expectedNodeID)
					}
					oddNode = !oddNode
				}
			}

			s.ElementsMatch(expectedNodeIds, fetchedNodeIDs)
		})
	}
}

func (s *nodeDatastoreSACSuite) TestUpsertNode() {
	clusterID := testconsts.Cluster2

	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), "upsert")
	for name, c := range cases {
		s.Run(name, func() {
			nodeID := uuid.NewV4().String()
			node := fixtures.GetScopedNode(nodeID, clusterID)

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertNode(ctx, node)

			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)

				// Check that node is not added to datastore.
				_, found, errGetNode := s.datastore.GetNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], node.GetId())
				s.False(found)
				s.NoError(errGetNode)
			} else {
				s.NoError(err)

				s.testNodeIDs[clusterID] = append(s.testNodeIDs[clusterID], nodeID)
				s.testNodes[nodeID] = node

				// Check that node is added in datastore.
				_, found, errGetNode := s.datastore.GetNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], node.GetId())
				s.True(found)
				s.NoError(errGetNode)
			}
		})
	}
}

func (s *nodeDatastoreSACSuite) TestDeleteNodesSingle() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())
	for name, c := range cases {
		s.Run(name, func() {
			var err error

			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			ctx := s.testContexts[c.ScopeKey]

			targetClusterID := testconsts.Cluster1
			delNodeID := s.addTestNode(targetClusterID)

			_, foundTestNode, err := s.datastore.GetNode(unrestrictedCtx, delNodeID)
			s.Require().True(foundTestNode)
			s.Require().NoError(err)

			err = s.datastore.DeleteNodes(ctx, delNodeID)
			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)

				// Check that node is still in datastore.
				_, found, errGetNode := s.datastore.GetNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], delNodeID)
				s.True(found)
				s.NoError(errGetNode)
			} else {
				s.NoError(err)

				// Check that node is removed from datastore.
				_, found, errGetNode := s.datastore.GetNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], delNodeID)
				s.False(found)
				s.NoError(errGetNode)

				// And ensure another sibling node is still there.
				_, found, errGetNode = s.datastore.GetNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], s.testNodeIDs[targetClusterID][0])
				s.True(found)
				s.NoError(errGetNode)
			}
		})
	}
}

func (s *nodeDatastoreSACSuite) TestDeleteNodesMulti() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())
	for name, c := range cases {
		s.Run(name, func() {
			var err error
			var foundTestNode bool

			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			ctx := s.testContexts[c.ScopeKey]

			targetClusterID := testconsts.Cluster2
			var delNodeIDs []string
			for i := 0; i < 3; i++ {
				testNodeID := s.addTestNode(targetClusterID)

				delNodeIDs = append(delNodeIDs, testNodeID)
			}

			for _, delNodeID := range delNodeIDs {
				_, foundTestNode, err = s.datastore.GetNode(unrestrictedCtx, delNodeID)
				s.Require().True(foundTestNode)
				s.Require().NoError(err)
			}

			err = s.datastore.DeleteNodes(ctx, delNodeIDs...)
			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)

				// Check that nodes are not removed from datastore.
				for _, delNodeID := range delNodeIDs {
					_, found, errGetNode := s.datastore.GetNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], delNodeID)
					s.True(found)
					s.NoError(errGetNode)
				}
			} else {
				s.NoError(err)

				// Check that nodes are removed from datastore.
				for _, delNodeID := range delNodeIDs {
					_, found, errGetNode := s.datastore.GetNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], delNodeID)
					s.False(found)
					s.NoError(errGetNode)
				}

				// And ensure another sibling node is still there.
				_, found, errGetNode := s.datastore.GetNode(s.testContexts[testutils.UnrestrictedReadWriteCtx], s.testNodeIDs[targetClusterID][0])
				s.True(found)
				s.NoError(errGetNode)
			}
		})
	}
}
