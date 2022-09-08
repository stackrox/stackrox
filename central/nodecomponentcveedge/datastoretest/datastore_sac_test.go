package datastoretest

import (
	"context"
	"testing"

	componentCVEEdgeDataStore "github.com/stackrox/rox/central/componentcveedge/datastore"
	"github.com/stackrox/rox/central/cve/converter/utils"
	dackboxTestUtils "github.com/stackrox/rox/central/dackbox/testutils"
	"github.com/stackrox/rox/central/nodecomponentcveedge/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/suite"
)

var (
	nodeScanOperatingSystem = "Linux"

	dontWaitForIndexing = false
	waitForIndexing     = true
)

func TestNodeComponentCVEEdgeDatastoreSAC(t *testing.T) {
	suite.Run(t, new(nodeComponentCVEEdgeDatastoreSACTestSuite))
}

type nodeComponentCVEEdgeDatastoreSACTestSuite struct {
	suite.Suite

	dackboxTestStore dackboxTestUtils.DackboxTestDataStore
	datastore        datastore.DataStore

	testContexts map[string]context.Context
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) SetupSuite() {
	var err error
	s.dackboxTestStore, err = dackboxTestUtils.NewDackboxTestDataStore(s.T())
	s.Require().NoError(err)
	if features.PostgresDatastore.Enabled() {
		pool := s.dackboxTestStore.GetPostgresPool()
		s.datastore, err = datastore.GetTestPostgresDataStore(s.T(), pool)
		s.Require().NoError(err)
	} else {
		rocksengine := s.dackboxTestStore.GetRocksEngine()
		bleveIndex := s.dackboxTestStore.GetBleveIndex()
		dacky := s.dackboxTestStore.GetDackbox()
		genericStore, err := componentCVEEdgeDataStore.GetTestRocksBleveDataStore(s.T(), rocksengine, bleveIndex, dacky)
		s.Require().NoError(err)
		s.datastore = nodeComponentCVEEdgeFromGenericStore{genericStore: genericStore}
	}
	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TearDownSuite() {
	s.Require().NoError(s.dackboxTestStore.Cleanup(s.T()))
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) cleanNodeToVulnerabilitiesGraph(waitForIndexing bool) {
	s.Require().NoError(s.dackboxTestStore.CleanNodeToVulnerabilitiesGraph(waitForIndexing))
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
	if features.PostgresDatastore.Enabled() {
		return postgres.IDFromPks([]string{componentID, vulnerabilityID})
	}
	return edges.EdgeID{ParentID: componentID, ChildID: vulnerabilityID}.ToString()
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
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2ReadWriteCtx,
			expectedEdgeFound: cluster2Map,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespacesABReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster3ReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// Has access to Cluster1 + NamespaceA as well as full access to Cluster2.
			expectedEdgeFound: cluster2Map,
		},
	}
)

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSingleComponent() {
	// Inject the fixture graph, and test exists for Component1 to CVE-1234-0001 edge
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp1cve1edge
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			exists, err := s.datastore.Exists(testCtx, targetEdgeID)
			s.NoError(err)
			s.Equal(c.expectedEdgeFound[targetEdgeID], exists)
		})
	}
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSingleComponentToSharedCVE() {
	// Inject the fixture graph, and test exists for Component1 to CVE-4567-0002 edge
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp1cve2edge
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			exists, err := s.datastore.Exists(testCtx, targetEdgeID)
			s.NoError(err)
			s.Equal(c.expectedEdgeFound[targetEdgeID], exists)
		})
	}
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSharedComponent() {
	// Inject the fixture graph, and test exists for Component3 to CVE-3456-0004 edge
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp3cve4edge
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			exists, err := s.datastore.Exists(testCtx, targetEdgeID)
			s.NoError(err)
			s.Equal(c.expectedEdgeFound[targetEdgeID], exists)
		})
	}
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSingleComponent() {
	// Inject the fixture graph, and test read for Component1 to CVE-1234-0001 edge
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp1cve1edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)
	expectedTargetID := getVulnerabilityID(fixtures.GetEmbeddedNodeCVE1234x0001(), nodeScanOperatingSystem)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
			s.NoError(err)
			if c.expectedEdgeFound[targetEdgeID] {
				s.True(exists)
				s.NotNil(edge)
				s.Equal(expectedSrcID, edge.GetNodeComponentId())
				s.Equal(expectedTargetID, edge.GetNodeCveId())
			} else {
				s.False(exists)
				s.Nil(edge)
			}
		})
	}
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSingleComponentToSharedCVE() {
	// Inject the fixture graph, and test read for Component1 to CVE-4567-0002 edge
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp1cve2edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedNodeComponent1x1(), nodeScanOperatingSystem)
	expectedTargetID := getVulnerabilityID(fixtures.GetEmbeddedNodeCVE4567x0002(), nodeScanOperatingSystem)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
			s.NoError(err)
			if c.expectedEdgeFound[targetEdgeID] {
				s.True(exists)
				s.NotNil(edge)
				s.Equal(expectedSrcID, edge.GetNodeComponentId())
				s.Equal(expectedTargetID, edge.GetNodeCveId())
			} else {
				s.False(exists)
				s.Nil(edge)
			}
		})
	}
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSharedComponent() {
	// Inject the fixture graph, and test read for Component3 to CVE-3456-0004 edge
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp3cve4edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedNodeComponent1s2x3(), nodeScanOperatingSystem)
	expectedTargetID := getVulnerabilityID(fixtures.GetEmbeddedNodeCVE3456x0004(), nodeScanOperatingSystem)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
			s.NoError(err)
			if c.expectedEdgeFound[targetEdgeID] {
				s.True(exists)
				s.NotNil(edge)
				s.Equal(expectedSrcID, edge.GetNodeComponentId())
				s.Equal(expectedTargetID, edge.GetNodeCveId())
			} else {
				s.False(exists)
				s.Nil(edge)
			}
		})
	}
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestCount() {
	// Inject the fixture graph, and test data filtering on count operations
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(waitForIndexing)
	s.Require().NoError(err)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
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
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestSearch() {
	// Inject the fixture graph, and test data filtering on count operations
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(waitForIndexing)
	s.Require().NoError(err)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			expectedCount := 0
			for _, visible := range c.expectedEdgeFound {
				if visible {
					expectedCount++
				}
			}
			testCtx := s.testContexts[c.contextKey]
			results, err := s.datastore.Search(testCtx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(expectedCount, len(results))
			for _, r := range results {
				s.True(c.expectedEdgeFound[r.ID])
			}
		})
	}
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestSearchEdges() {
	// Inject the fixture graph, and test data filtering on count operations
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(waitForIndexing)
	s.Require().NoError(err)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			expectedCount := 0
			for _, visible := range c.expectedEdgeFound {
				if visible {
					expectedCount++
				}
			}
			testCtx := s.testContexts[c.contextKey]
			results, err := s.datastore.SearchEdges(testCtx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(expectedCount, len(results))
			for _, r := range results {
				s.True(c.expectedEdgeFound[r.GetId()])
			}
		})
	}
}

func (s *nodeComponentCVEEdgeDatastoreSACTestSuite) TestSearchRawEdges() {
	// Inject the fixture graph, and test data filtering on count operations
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanNodeToVulnerabilitiesGraph(waitForIndexing)
	s.Require().NoError(err)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			expectedCount := 0
			for _, visible := range c.expectedEdgeFound {
				if visible {
					expectedCount++
				}
			}
			testCtx := s.testContexts[c.contextKey]
			results, err := s.datastore.SearchRawEdges(testCtx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(expectedCount, len(results))
			for _, r := range results {
				s.True(c.expectedEdgeFound[r.GetId()])
			}
		})
	}
}
