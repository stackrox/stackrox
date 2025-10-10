//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/suite"
)

func TestNodeComponentDataStoreSAC(t *testing.T) {
	if features.FlattenCVEData.Enabled() {
		t.Skip("FlattenCVEData enabled.  Test is obsolete.")
	}
	suite.Run(t, new(cveDataStoreSACTestSuite))
}

type cveDataStoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore graphDBTestUtils.TestGraphDataStore
	nodeComponentStore DataStore

	nodeTestContexts map[string]context.Context
}

func (s *cveDataStoreSACTestSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.nodeComponentStore = GetTestPostgresDataStore(s.T(), pool)
	s.nodeTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
}

func getNodeComponentID(component *storage.EmbeddedNodeScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func (s *cveDataStoreSACTestSuite) cleanNodeToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanNodeToVulnerabilitiesGraph())
}

type componentTestCase struct {
	contextKey             string
	expectedComponentFound map[string]bool
}

var (
	node1OS              = fixtures.GetScopedNode1("dummyNodeID", testconsts.Cluster1).GetScan().GetOperatingSystem()
	node2OS              = fixtures.GetScopedNode1("dummyNodeID", testconsts.Cluster2).GetScan().GetOperatingSystem()
	nodeComponent1x1     = fixtures.GetEmbeddedNodeComponent1x1()
	nodeComponent1x2     = fixtures.GetEmbeddedNodeComponent1x2()
	nodeComponent1s2x3   = fixtures.GetEmbeddedNodeComponent1s2x3()
	nodeComponent2x4     = fixtures.GetEmbeddedNodeComponent2x4()
	nodeComponent2x5     = fixtures.GetEmbeddedNodeComponent2x5()
	nodeComponentID1x1   = getNodeComponentID(nodeComponent1x1, node1OS)
	nodeComponentID1x2   = getNodeComponentID(nodeComponent1x2, node1OS)
	nodeComponentID1s2x3 = getNodeComponentID(nodeComponent1s2x3, node1OS)
	nodeComponentID2x4   = getNodeComponentID(nodeComponent2x4, node2OS)
	nodeComponentID2x5   = getNodeComponentID(nodeComponent2x5, node2OS)

	nodeComponentTestCases = []componentTestCase{
		{
			contextKey: sacTestUtils.UnrestrictedReadCtx,
			expectedComponentFound: map[string]bool{
				nodeComponentID1x1:   true,
				nodeComponentID1x2:   true,
				nodeComponentID1s2x3: true,
				nodeComponentID2x4:   true,
				nodeComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.UnrestrictedReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponentID1x1:   true,
				nodeComponentID1x2:   true,
				nodeComponentID1s2x3: true,
				nodeComponentID2x4:   true,
				nodeComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponentID1x1:   true,
				nodeComponentID1x2:   true,
				nodeComponentID1s2x3: true,
				nodeComponentID2x4:   false,
				nodeComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponentID1x1:   true,
				nodeComponentID1x2:   true,
				nodeComponentID1s2x3: true,
				nodeComponentID2x4:   false,
				nodeComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponentID1x1:   false,
				nodeComponentID1x2:   false,
				nodeComponentID1s2x3: true,
				nodeComponentID2x4:   true,
				nodeComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponentID1x1:   false,
				nodeComponentID1x2:   false,
				nodeComponentID1s2x3: true,
				nodeComponentID2x4:   true,
				nodeComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponentID1x1:   false,
				nodeComponentID1x2:   false,
				nodeComponentID1s2x3: false,
				nodeComponentID2x4:   false,
				nodeComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
			// Therefore it should see only cluster2 vulnerabilities (and shared ones).
			expectedComponentFound: map[string]bool{
				nodeComponentID1x1:   true,
				nodeComponentID1x2:   true,
				nodeComponentID1s2x3: true,
				nodeComponentID2x4:   true,
				nodeComponentID2x5:   true,
			},
		},
	}
)

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for Component1x1
	s.runNodeTest("TestSACNodeComponentExistsSingleScopeOnly", func(c componentTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		exists, err := s.nodeComponentStore.Exists(testCtx, nodeComponentID1x1)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[nodeComponentID1x1], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentExistsSharedComponent() {
	// Inject the fixture graph, and test exists for Component1s2x3
	s.runNodeTest("TestSACNodeComponentExistsSharedComponent", func(c componentTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		exists, err := s.nodeComponentStore.Exists(testCtx, nodeComponentID1s2x3)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[nodeComponentID1s2x3], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for Component1x1
	targetComponent := nodeComponent1x1
	componentName := targetComponent.GetName()
	cvss := targetComponent.GetTopCvss()
	s.runNodeTest("TestSACNodeComponentGetSingleScopeOnly", func(c componentTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		nodeComponent, found, err := s.nodeComponentStore.Get(testCtx, nodeComponentID1x1)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[nodeComponentID1x1], found)
		if c.expectedComponentFound[nodeComponentID1x1] {
			s.NotNil(nodeComponent)
			s.Equal(componentName, nodeComponent.GetName())
			s.Equal(cvss, nodeComponent.GetTopCvss())
		} else {
			s.Nil(nodeComponent)
		}
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentGetSharedComponent() {
	// Inject the fixture graph, and test retrieval for Component1s2x3
	targetComponent := nodeComponent1s2x3
	componentName := targetComponent.GetName()
	cvss := targetComponent.GetTopCvss()
	s.runNodeTest("TestSACNodeComponentGetSharedComponent", func(c componentTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		nodeComponent, found, err := s.nodeComponentStore.Get(testCtx, nodeComponentID1s2x3)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[nodeComponentID1s2x3], found)
		if c.expectedComponentFound[nodeComponentID1s2x3] {
			s.NotNil(nodeComponent)
			s.Equal(componentName, nodeComponent.GetName())
			s.Equal(cvss, nodeComponent.GetTopCvss())
		} else {
			s.Nil(nodeComponent)
		}
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentGetBatch() {
	componentIDs := []string{
		nodeComponentID1x1,
		nodeComponentID1x2,
		nodeComponentID1s2x3,
		nodeComponentID2x5,
	}
	s.runNodeTest("TestSACNodeComponentGetBatch", func(c componentTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		nodeComponents, err := s.nodeComponentStore.GetBatch(testCtx, componentIDs)
		s.NoError(err)
		expectedComponentIDs := make([]string, 0, len(componentIDs))
		for _, componentID := range componentIDs {
			if c.expectedComponentFound[componentID] {
				expectedComponentIDs = append(expectedComponentIDs, componentID)
			}
		}
		fetchedComponentIDs := make([]string, 0, len(nodeComponents))
		for _, nodeComponent := range nodeComponents {
			fetchedComponentIDs = append(fetchedComponentIDs, nodeComponent.GetId())
		}
		s.ElementsMatch(expectedComponentIDs, fetchedComponentIDs)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentCount() {
	s.T().Skip("Skipping Component count tests for now.")
	s.runNodeTest("TestSACNodeComponentCount", func(c componentTestCase) {

		testCtx := s.nodeTestContexts[c.contextKey]
		count, err := s.nodeComponentStore.Count(testCtx, nil)
		s.NoError(err)
		expectedCount := 0
		for _, visible := range c.expectedComponentFound {
			if visible {
				expectedCount++
			}
		}
		s.Equal(expectedCount, count)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentSearch() {
	s.runNodeTest("TestSACNodeComponentSearch", func(c componentTestCase) {

		testCtx := s.nodeTestContexts[c.contextKey]
		results, err := s.nodeComponentStore.Search(testCtx, nil)
		s.NoError(err)
		expectedComponentIDs := make([]string, 0, len(c.expectedComponentFound))
		for ID, visible := range c.expectedComponentFound {
			if visible {
				expectedComponentIDs = append(expectedComponentIDs, ID)
			}
		}
		fetchedComponentIDset := make(map[string]bool, 0)
		for _, result := range results {
			fetchedComponentIDset[result.ID] = true
		}
		fetchedComponentIDs := make([]string, 0, len(fetchedComponentIDset))
		for id := range fetchedComponentIDset {
			fetchedComponentIDs = append(fetchedComponentIDs, id)
		}
		s.ElementsMatch(fetchedComponentIDs, expectedComponentIDs)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentSearchNodeComponents() {
	s.runNodeTest("TestSACNodeComponentSearchNodeComponents", func(c componentTestCase) {

		testCtx := s.nodeTestContexts[c.contextKey]
		results, err := s.nodeComponentStore.SearchNodeComponents(testCtx, nil)
		s.NoError(err)
		expectedComponentIDs := make([]string, 0, len(c.expectedComponentFound))
		for ID, visible := range c.expectedComponentFound {
			if visible {
				expectedComponentIDs = append(expectedComponentIDs, ID)
			}
		}
		fetchedComponentIDset := make(map[string]bool, 0)
		for _, result := range results {
			fetchedComponentIDset[result.GetId()] = true
		}
		fetchedComponentIDs := make([]string, 0, len(fetchedComponentIDset))
		for id := range fetchedComponentIDset {
			fetchedComponentIDs = append(fetchedComponentIDs, id)
		}
		s.ElementsMatch(fetchedComponentIDs, expectedComponentIDs)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentSearchRawNodeComponents() {
	s.runNodeTest("TestSACNodeComponentSearchRawNodeComponents", func(c componentTestCase) {

		testCtx := s.nodeTestContexts[c.contextKey]
		results, err := s.nodeComponentStore.SearchRawNodeComponents(testCtx, nil)
		s.NoError(err)
		expectedComponentIDs := make([]string, 0, len(c.expectedComponentFound))
		for ID, visible := range c.expectedComponentFound {
			if visible {
				expectedComponentIDs = append(expectedComponentIDs, ID)
			}
		}
		fetchedComponentIDset := make(map[string]bool, 0)
		for _, result := range results {
			fetchedComponentIDset[result.GetId()] = true
		}
		fetchedComponentIDs := make([]string, 0, len(fetchedComponentIDset))
		for id := range fetchedComponentIDset {
			fetchedComponentIDs = append(fetchedComponentIDs, id)
		}
		s.ElementsMatch(fetchedComponentIDs, expectedComponentIDs)
	})
}

func (s *cveDataStoreSACTestSuite) runNodeTest(testName string, testFunc func(c componentTestCase)) {
	err := s.testGraphDatastore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	s.Require().NoError(err)
	nodeGraphBefore := graphDBTestUtils.GetNodeGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false
	for _, c := range nodeComponentTestCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			// When triggered in parallel, most tests fail.
			// TearDownTest is executed before the sub-tests.
			// See https://github.com/stretchr/testify/issues/934
			// s.T().Parallel()
			testFunc(c)
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		s.T().Logf("%s failed, dumping DB content.", testName)
		nodeGraphBefore.Log()
	}
}
