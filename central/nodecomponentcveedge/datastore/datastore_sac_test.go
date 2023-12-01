//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/converter/utils"
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

var (
	nodeScanOperatingSystem = "Linux"

	log = logging.LoggerForModule()
)

func TestNodeComponentCVEEdgeDatastoreSAC(t *testing.T) {
	suite.Run(t, new(nodeComponentCVEEdgeDatastoreSACTestSuite))
}

type nodeComponentCVEEdgeDatastoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore graphDBTestUtils.TestGraphDataStore
	datastore          DataStore

	testContexts map[string]context.Context
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)

	pool := s.testGraphDatastore.GetPostgresPool()
	s.datastore = GetTestPostgresDataStore(s.T(), pool)
	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)

	err = s.testGraphDatastore.PushNodeToVulnerabilitiesGraph()
	s.Require().NoError(err)
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

func getComponentID(component *storage.EmbeddedNodeScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func getVulnerabilityID(vulnerability *storage.EmbeddedVulnerability, os string) string {
	return utils.EmbeddedCVEToProtoCVE(os, vulnerability).GetId()
}

func getEdgeID(component *storage.EmbeddedNodeScanComponent, vulnerability *storage.EmbeddedVulnerability, os string) string {
	componentID := getComponentID(component, os)
	vulnerabilityID := getVulnerabilityID(vulnerability, os)
	return pgSearch.IDFromPks([]string{componentID, vulnerabilityID})
}

type edgeTestCase struct {
	contextKey        string
	expectedEdgeFound map[string]bool
}

var (
	cmp1cve1edge = getEdgeID(fixtures.GetEmbeddedNodeComponent1x1(), fixtures.GetEmbeddedNodeCVE1234x0001(), nodeScanOperatingSystem)
	cmp1cve2edge = getEdgeID(fixtures.GetEmbeddedNodeComponent1x1(), fixtures.GetEmbeddedNodeCVE4567x0002(), nodeScanOperatingSystem)
	cmp2cve3edge = getEdgeID(fixtures.GetEmbeddedNodeComponent1x2(), fixtures.GetEmbeddedNodeCVE1234x0003(), nodeScanOperatingSystem)
	cmp3cve4edge = getEdgeID(fixtures.GetEmbeddedNodeComponent1s2x3(), fixtures.GetEmbeddedNodeCVE3456x0004(), nodeScanOperatingSystem)
	cmp3cve5edge = getEdgeID(fixtures.GetEmbeddedNodeComponent1s2x3(), fixtures.GetEmbeddedNodeCVE3456x0005(), nodeScanOperatingSystem)
	cmp5cve2edge = getEdgeID(fixtures.GetEmbeddedNodeComponent2x5(), fixtures.GetEmbeddedNodeCVE4567x0002(), nodeScanOperatingSystem)
	cmp5cve6edge = getEdgeID(fixtures.GetEmbeddedNodeComponent2x5(), fixtures.GetEmbeddedNodeCVE2345x0006(), nodeScanOperatingSystem)
	cmp5cve7edge = getEdgeID(fixtures.GetEmbeddedNodeComponent2x5(), fixtures.GetEmbeddedNodeCVE2345x0007(), nodeScanOperatingSystem)

	fullAccessMap = map[string]bool{
		cmp1cve1edge: true,
		cmp1cve2edge: true,
		cmp2cve3edge: true,
		cmp3cve4edge: true,
		cmp3cve5edge: true,
		cmp5cve2edge: true,
		cmp5cve6edge: true,
		cmp5cve7edge: true,
	}

	cluster1Map = map[string]bool{
		cmp1cve1edge: true,
		cmp1cve2edge: true,
		cmp2cve3edge: true,
		cmp3cve4edge: true,
		cmp3cve5edge: true,
		cmp5cve2edge: false,
		cmp5cve6edge: false,
		cmp5cve7edge: false,
	}

	cluster2Map = map[string]bool{
		cmp1cve1edge: false,
		cmp1cve2edge: false,
		cmp2cve3edge: false,
		cmp3cve4edge: true,
		cmp3cve5edge: true,
		cmp5cve2edge: true,
		cmp5cve6edge: true,
		cmp5cve7edge: true,
	}

	noAccessMap = map[string]bool{
		cmp1cve1edge: false,
		cmp1cve2edge: false,
		cmp2cve3edge: false,
		cmp3cve4edge: false,
		cmp3cve5edge: false,
		cmp5cve2edge: false,
		cmp5cve6edge: false,
		cmp5cve7edge: false,
	}

	testCases = []edgeTestCase{
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
			expectedEdgeFound: cluster1Map,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedEdgeFound: cluster1Map,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedEdgeFound: cluster1Map,
		},
		{
			contextKey:        sacTestUtils.Cluster2ReadWriteCtx,
			expectedEdgeFound: cluster2Map,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedEdgeFound: cluster2Map,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespacesABReadWriteCtx,
			expectedEdgeFound: cluster2Map,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedEdgeFound: cluster2Map,
		},
		{
			contextKey:        sacTestUtils.Cluster3ReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// Has access to Cluster1 + NamespaceA as well as full access to Cluster2.
			expectedEdgeFound: fullAccessMap,
		},
	}
)

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSingleComponent() {
	// Inject the fixture graph, and test exists for Component1 to CVE-1234-0001 edge
	targetEdgeID := cmp1cve1edge
	s.run("TestExistsEdgeFromSingleComponent", func(c edgeTestCase) {
		testCtx := s.testContexts[c.contextKey]
		exists, err := s.datastore.Exists(testCtx, targetEdgeID)
		s.NoError(err)
		s.Equal(c.expectedEdgeFound[targetEdgeID], exists)
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSingleComponentToSharedCVE() {
	// Inject the fixture graph, and test exists for Component1 to CVE-4567-0002 edge
	targetEdgeID := cmp1cve2edge
	s.run("TestExistsEdgeFromSingleComponentToSharedCVE", func(c edgeTestCase) {
		testCtx := s.testContexts[c.contextKey]
		exists, err := s.datastore.Exists(testCtx, targetEdgeID)
		s.NoError(err)
		s.Equal(c.expectedEdgeFound[targetEdgeID], exists)
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSharedComponent() {
	// Inject the fixture graph, and test exists for Component3 to CVE-3456-0004 edge
	targetEdgeID := cmp3cve4edge
	s.run("TestExistsEdgeFromSharedComponent", func(c edgeTestCase) {
		testCtx := s.testContexts[c.contextKey]
		exists, err := s.datastore.Exists(testCtx, targetEdgeID)
		s.NoError(err)
		s.Equal(c.expectedEdgeFound[targetEdgeID], exists)
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSingleComponent() {
	// Inject the fixture graph, and test read for Component1 to CVE-1234-0001 edge
	targetEdgeID := cmp1cve1edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)
	expectedTargetID := getVulnerabilityID(fixtures.GetEmbeddedNodeCVE1234x0001(), nodeScanOperatingSystem)
	s.run("TestGetEdgeFromSingleComponent", func(c edgeTestCase) {
		testCtx := s.testContexts[c.contextKey]
		edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
		s.NoError(err)
		if c.expectedEdgeFound[targetEdgeID] {
			s.True(exists)
			s.Require().NotNil(edge)
			s.Equal(expectedSrcID, edge.GetNodeComponentId())
			s.Equal(expectedTargetID, edge.GetNodeCveId())
		} else {
			s.False(exists)
			s.Nil(edge)
		}
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSingleComponentToSharedCVE() {
	// Inject the fixture graph, and test read for Component1 to CVE-4567-0002 edge
	targetEdgeID := cmp1cve2edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)
	expectedTargetID := getVulnerabilityID(fixtures.GetEmbeddedNodeCVE4567x0002(), nodeScanOperatingSystem)
	s.run("TestGetEdgeFromSingleComponentToSharedCVE", func(c edgeTestCase) {
		testCtx := s.testContexts[c.contextKey]
		edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
		s.NoError(err)
		if c.expectedEdgeFound[targetEdgeID] {
			s.True(exists)
			s.Require().NotNil(edge)
			s.Equal(expectedSrcID, edge.GetNodeComponentId())
			s.Equal(expectedTargetID, edge.GetNodeCveId())
		} else {
			s.False(exists)
			s.Nil(edge)
		}
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSharedComponent() {
	// Inject the fixture graph, and test read for Component3 to CVE-3456-0004 edge
	targetEdgeID := cmp3cve4edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedNodeComponent1s2x3(), nodeScanOperatingSystem)
	expectedTargetID := getVulnerabilityID(fixtures.GetEmbeddedNodeCVE3456x0004(), nodeScanOperatingSystem)
	s.run("TestGetEdgeFromSharedComponent", func(c edgeTestCase) {
		testCtx := s.testContexts[c.contextKey]
		edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
		s.NoError(err)
		if c.expectedEdgeFound[targetEdgeID] {
			s.True(exists)
			s.Require().NotNil(edge)
			s.Equal(expectedSrcID, edge.GetNodeComponentId())
			s.Equal(expectedTargetID, edge.GetNodeCveId())
		} else {
			s.False(exists)
			s.Nil(edge)
		}
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestCount() {
	// Inject the fixture graph, and test data filtering on count operations
	s.run("TestCount", func(c edgeTestCase) {
		expectedCount := 0
		for _, visible := range c.expectedEdgeFound {
			if visible {
				expectedCount++
			}
		}
		testCtx := s.testContexts[c.contextKey]
		count, err := s.datastore.Count(testCtx, search.EmptyQuery())
		s.NoError(err)
		s.Equal(expectedCount, count)
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestSearch() {
	// Inject the fixture graph, and test data filtering on count operations
	s.run("TestSearch", func(c edgeTestCase) {
		expectedCount := 0
		for _, visible := range c.expectedEdgeFound {
			if visible {
				expectedCount++
			}
		}
		testCtx := s.testContexts[c.contextKey]
		results, err := s.datastore.Search(testCtx, search.EmptyQuery())
		s.NoError(err)
		s.Len(results, expectedCount)
		for _, r := range results {
			s.True(c.expectedEdgeFound[r.ID])
		}
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestSearchEdges() {
	// Inject the fixture graph, and test data filtering on count operations
	s.run("TestSearchEdges", func(c edgeTestCase) {
		expectedCount := 0
		for _, visible := range c.expectedEdgeFound {
			if visible {
				expectedCount++
			}
		}
		testCtx := s.testContexts[c.contextKey]
		results, err := s.datastore.SearchEdges(testCtx, search.EmptyQuery())
		s.NoError(err)
		s.Len(results, expectedCount)
		for _, r := range results {
			s.True(c.expectedEdgeFound[r.GetId()])
		}
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestSearchRawEdges() {
	// Inject the fixture graph, and test data filtering on count operations
	s.run("TestSearchRawEdges", func(c edgeTestCase) {
		expectedCount := 0
		for _, visible := range c.expectedEdgeFound {
			if visible {
				expectedCount++
			}
		}
		testCtx := s.testContexts[c.contextKey]
		results, err := s.datastore.SearchRawEdges(testCtx, search.EmptyQuery())
		s.NoError(err)
		s.Len(results, expectedCount)
		for _, r := range results {
			s.True(c.expectedEdgeFound[r.GetId()])
		}
	})
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) run(testName string, f func(c edgeTestCase)) {
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false
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
