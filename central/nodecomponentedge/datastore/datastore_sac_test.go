//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/suite"
)

const (
	nodeScanOperatingSystem = "Linux"
)

var (
	log = logging.LoggerForModule()
)

func TestNodeComponentEdgeDatastoreSAC(t *testing.T) {
	suite.Run(t, new(nodeComponentEdgeDatastoreSACTestSuite))
}

type nodeComponentEdgeDatastoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore graphDBTestUtils.TestGraphDataStore
	datastore          DataStore

	testContexts map[string]context.Context
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.datastore, err = GetTestPostgresDataStore(s.T(), pool)
	s.Require().NoError(err)
	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
	err = s.testGraphDatastore.PushNodeToVulnerabilitiesGraph()
	s.Require().NoError(err)
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

func getComponentID(component *storage.EmbeddedNodeScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func getEdgeID(nodeID string, component *storage.EmbeddedNodeScanComponent, os string) string {
	componentID := getComponentID(component, os)
	return pgSearch.IDFromPks([]string{nodeID, componentID})
}

type edgeTestCase struct {
	contextKey        string
	expectedEdgeFound map[string]bool
}

func getTestCases(nodeIDs []string) []edgeTestCase {
	node1 := nodeIDs[0]
	node2 := nodeIDs[1]

	node1cmp1edge := getEdgeID(node1, fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)
	node1cmp2edge := getEdgeID(node1, fixtures.GetEmbeddedNodeComponent1x2(), nodeScanOperatingSystem)
	node1cmp3edge := getEdgeID(node1, fixtures.GetEmbeddedNodeComponent1s2x3(), nodeScanOperatingSystem)
	node2cmp3edge := getEdgeID(node2, fixtures.GetEmbeddedNodeComponent1s2x3(), nodeScanOperatingSystem)
	node2cmp4edge := getEdgeID(node2, fixtures.GetEmbeddedNodeComponent2x4(), nodeScanOperatingSystem)
	node2cmp5edge := getEdgeID(node2, fixtures.GetEmbeddedNodeComponent2x5(), nodeScanOperatingSystem)

	fullAccessMap := map[string]bool{
		node1cmp1edge: true,
		node1cmp2edge: true,
		node1cmp3edge: true,
		node2cmp3edge: true,
		node2cmp4edge: true,
		node2cmp5edge: true,
	}
	cluster1AccessMap := map[string]bool{
		node1cmp1edge: true,
		node1cmp2edge: true,
		node1cmp3edge: true,
		node2cmp3edge: false,
		node2cmp4edge: false,
		node2cmp5edge: false,
	}
	cluster2AccessMap := map[string]bool{
		node1cmp1edge: false,
		node1cmp2edge: false,
		node1cmp3edge: false,
		node2cmp3edge: true,
		node2cmp4edge: true,
		node2cmp5edge: true,
	}
	noAccessMap := map[string]bool{
		node1cmp1edge: false,
		node1cmp2edge: false,
		node1cmp3edge: false,
		node2cmp3edge: false,
		node2cmp4edge: false,
		node2cmp5edge: false,
	}

	testCases := []edgeTestCase{
		{
			contextKey:        sacTestUtils.UnrestrictedReadCtx,
			expectedEdgeFound: fullAccessMap,
		},
		{
			contextKey:        sacTestUtils.UnrestrictedReadWriteCtx,
			expectedEdgeFound: fullAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1ReadWriteCtx,
			expectedEdgeFound: cluster1AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedEdgeFound: cluster1AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedEdgeFound: cluster1AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedEdgeFound: cluster1AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedEdgeFound: cluster1AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2ReadWriteCtx,
			expectedEdgeFound: cluster2AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedEdgeFound: cluster2AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedEdgeFound: cluster2AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespacesABReadWriteCtx,
			expectedEdgeFound: cluster2AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedEdgeFound: cluster2AccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster3ReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// Has access to Cluster1 + NamespaceA as well as full access to Cluster2 (including NamespaceB).
			expectedEdgeFound: fullAccessMap,
		},
	}
	return testCases
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) TestExists() {
	// Inject the fixture graph and test for node1 to component1 edge
	s.runReadTest("TestExists", func(c edgeTestCase, nodeIDs []string) {
		node1 := nodeIDs[0]
		targetEdgeID := getEdgeID(node1, fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)

		ctx := s.testContexts[c.contextKey]
		found, err := s.datastore.Exists(ctx, targetEdgeID)
		s.NoError(err)
		s.Equal(c.expectedEdgeFound[targetEdgeID], found)
	})
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) TestGet() {
	// Inject the fixture graph and test for node1 to component1 edge
	s.runReadTest("TestGet", func(c edgeTestCase, nodeIDs []string) {
		node1 := nodeIDs[0]
		targetEdgeID := getEdgeID(node1, fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)
		expectedSrcID := node1
		expectedTgtID := getComponentID(fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)

		ctx := s.testContexts[c.contextKey]
		obj, found, err := s.datastore.Get(ctx, targetEdgeID)
		s.NoError(err)
		if c.expectedEdgeFound[targetEdgeID] {
			s.True(found)
			s.Require().NotNil(obj)
			s.Equal(targetEdgeID, obj.GetId())
			s.Equal(expectedSrcID, obj.GetNodeId())
			s.Equal(expectedTgtID, obj.GetNodeComponentId())
		} else {
			s.False(found)
			s.Nil(obj)
		}
	})
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) TestGetBatch() {
	// Inject the fixture graph and test for node1 to component1 edge and node2 to component 4 edges
	s.runReadTest("TestGetBatch", func(c edgeTestCase, nodeIDs []string) {
		node1 := nodeIDs[0]
		targetEdge1ID := getEdgeID(node1, fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)
		expectedSrc1ID := node1
		expectedTgt1ID := getComponentID(fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)
		node2 := nodeIDs[1]
		targetEdge2ID := getEdgeID(node2, fixtures.GetEmbeddedNodeComponent2x4(), nodeScanOperatingSystem)
		expectedSrc2ID := node2
		expectedTgt2ID := getComponentID(fixtures.GetEmbeddedNodeComponent2x4(), nodeScanOperatingSystem)
		targetIDs := []string{targetEdge1ID, targetEdge2ID}

		ctx := s.testContexts[c.contextKey]
		edges, err := s.datastore.GetBatch(ctx, targetIDs)
		s.NoError(err)
		expectedEdgeCount := 0
		expectedEdge1 := false
		if c.expectedEdgeFound[targetEdge1ID] {
			expectedEdge1 = true
			expectedEdgeCount++
		}
		expectedEdge2 := false
		if c.expectedEdgeFound[targetEdge2ID] {
			expectedEdge2 = true
			expectedEdgeCount++
		}
		s.Len(edges, expectedEdgeCount)
		foundEdge1 := false
		foundEdge2 := false
		for _, e := range edges {
			edgeID := e.GetId()
			s.True(c.expectedEdgeFound[edgeID])
			if edgeID == targetEdge1ID {
				foundEdge1 = true
				s.Equal(expectedSrc1ID, e.GetNodeId())
				s.Equal(expectedTgt1ID, e.GetNodeComponentId())
			}
			if edgeID == targetEdge2ID {
				foundEdge2 = true
				s.Equal(expectedSrc2ID, e.GetNodeId())
				s.Equal(expectedTgt2ID, e.GetNodeComponentId())
			}
		}
		s.Equal(expectedEdge1, foundEdge1)
		s.Equal(expectedEdge2, foundEdge2)
	})
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) runReadTest(testName string, testFunc func(c edgeTestCase, nodeIDs []string)) {
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false

	nodeIDs := s.testGraphDatastore.GetStoredNodeIDs()
	testCases := getTestCases(nodeIDs)

	for i := range testCases {
		c := testCases[i]
		caseSucceeded := s.Run(c.contextKey, func() {
			// When triggered in parallel,
			// TearDownTest is executed before the sub-tests.
			// See https://github.com/stretchr/testify/issues/934
			// s.T().Parallel()
			testFunc(c, nodeIDs)
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Infof("%s failed, dumping DB content.", testName)
		imageGraphBefore.Log()
	}
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) TestCount() {
	s.run("TestCount", func(c edgeTestCase) {
		ctx := s.testContexts[c.contextKey]
		expectedCount := 0
		for _, visible := range c.expectedEdgeFound {
			if visible {
				expectedCount++
			}
		}
		count, err := s.datastore.Count(ctx)
		s.NoError(err)
		s.Equal(expectedCount, count)
	})
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) TestSearch() {
	s.run("TestSearch", func(c edgeTestCase) {
		ctx := s.testContexts[c.contextKey]
		expectedIDs := make([]string, 0, len(c.expectedEdgeFound))
		for edgeID, visible := range c.expectedEdgeFound {
			if visible {
				expectedIDs = append(expectedIDs, edgeID)
			}
		}
		fetchedIDs := make([]string, 0, len(c.expectedEdgeFound))
		res, err := s.datastore.Search(ctx, search.EmptyQuery())
		s.NoError(err)
		for _, r := range res {
			fetchedIDs = append(fetchedIDs, r.ID)
			s.True(c.expectedEdgeFound[r.ID])
		}
		s.ElementsMatch(expectedIDs, fetchedIDs)
	})
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) TestSearchEdges() {
	s.run("TestSearchEdges", func(c edgeTestCase) {
		ctx := s.testContexts[c.contextKey]
		expectedIDs := make([]string, 0, len(c.expectedEdgeFound))
		for edgeID, visible := range c.expectedEdgeFound {
			if visible {
				expectedIDs = append(expectedIDs, edgeID)
			}
		}
		fetchedIDs := make([]string, 0, len(c.expectedEdgeFound))
		res, err := s.datastore.SearchEdges(ctx, search.EmptyQuery())
		s.NoError(err)
		for _, r := range res {
			fetchedIDs = append(fetchedIDs, r.GetId())
			s.True(c.expectedEdgeFound[r.GetId()])
		}
		s.ElementsMatch(expectedIDs, fetchedIDs)
	})
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) TestSearchRawEdges() {
	s.run("TestSearchRawEdges", func(c edgeTestCase) {
		ctx := s.testContexts[c.contextKey]
		expectedIDs := make([]string, 0, len(c.expectedEdgeFound))
		for edgeID, visible := range c.expectedEdgeFound {
			if visible {
				expectedIDs = append(expectedIDs, edgeID)
			}
		}
		fetchedIDs := make([]string, 0, len(c.expectedEdgeFound))
		res, err := s.datastore.SearchRawEdges(ctx, search.EmptyQuery())
		s.NoError(err)
		for _, r := range res {
			fetchedIDs = append(fetchedIDs, r.GetId())
			s.True(c.expectedEdgeFound[r.GetId()])
		}
		s.ElementsMatch(expectedIDs, fetchedIDs)
	})
}

func (s *nodeComponentEdgeDatastoreSACTestSuite) run(testName string, f func(c edgeTestCase)) {
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false
	nodeIDs := s.testGraphDatastore.GetStoredNodeIDs()
	testCases := getTestCases(nodeIDs)
	for i := range testCases {
		c := testCases[i]
		caseSucceeded := s.Run(c.contextKey, func() {
			// When triggered in parallel,
			// TearDownTest is executed before the sub-tests.
			// See https://github.com/stretchr/testify/issues/934
			// s.T().Parallel()
			f(c)
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Infof("%s failed, dumping DB content.", testName)
		imageGraphBefore.Log()
	}
}
