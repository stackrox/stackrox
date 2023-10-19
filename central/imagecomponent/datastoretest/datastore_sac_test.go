//go:build sql_integration

package datastoretest

import (
	"context"
	"testing"

	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	imageComponentDataStore "github.com/stackrox/rox/central/imagecomponent/datastore"
	nodeComponentDataStore "github.com/stackrox/rox/central/nodecomponent/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/suite"
)

var (
	log = logging.LoggerForModule()
)

func TestImageComponentDataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveDataStoreSACTestSuite))
}

type cveDataStoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore  graphDBTestUtils.TestGraphDataStore
	imageComponentStore imageComponentDataStore.DataStore
	nodeComponentStore  nodeComponentDataStore.DataStore

	nodeTestContexts  map[string]context.Context
	imageTestContexts map[string]context.Context
}

func (s *cveDataStoreSACTestSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.imageComponentStore = imageComponentDataStore.GetTestPostgresDataStore(s.T(), pool)
	s.nodeComponentStore = nodeComponentDataStore.GetTestPostgresDataStore(s.T(), pool)
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
	s.nodeTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
}

func (s *cveDataStoreSACTestSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

func getImageComponentID(component *storage.EmbeddedImageScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func getNodeComponentID(component *storage.EmbeddedNodeScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func (s *cveDataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanImageToVulnerabilitiesGraph())
}

func (s *cveDataStoreSACTestSuite) cleanNodeToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanNodeToVulnerabilitiesGraph())
}

type componentTestCase struct {
	contextKey             string
	expectedComponentFound map[string]bool
}

var (
	image1OS              = fixtures.GetImageSherlockHolmes1().GetScan().GetOperatingSystem()
	image2OS              = fixtures.GetImageDoctorJekyll2().GetScan().GetOperatingSystem()
	imageComponent1x1     = fixtures.GetEmbeddedImageComponent1x1()
	imageComponent1x2     = fixtures.GetEmbeddedImageComponent1x2()
	imageComponent1s2x3   = fixtures.GetEmbeddedImageComponent1s2x3()
	imageComponent2x4     = fixtures.GetEmbeddedImageComponent2x4()
	imageComponent2x5     = fixtures.GetEmbeddedImageComponent2x5()
	imageComponentID1x1   = getImageComponentID(imageComponent1x1, image1OS)
	imageComponentID1x2   = getImageComponentID(imageComponent1x2, image1OS)
	imageComponentID1s2x3 = getImageComponentID(imageComponent1s2x3, image1OS)
	imageComponentID2x4   = getImageComponentID(imageComponent2x4, image2OS)
	imageComponentID2x5   = getImageComponentID(imageComponent2x5, image2OS)

	imageComponentTestCases = []componentTestCase{
		{
			contextKey: sacTestUtils.UnrestrictedReadCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   true,
				imageComponentID1x2:   true,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   true,
				imageComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.UnrestrictedReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   true,
				imageComponentID1x2:   true,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   true,
				imageComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   true,
				imageComponentID1x2:   true,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   true,
				imageComponentID1x2:   true,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: false,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   true,
				imageComponentID1x2:   true,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x2:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: false,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   true,
				imageComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: false,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   true,
				imageComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: false,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   true,
				imageComponentID2x5:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: false,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: false,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: false,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   false,
				imageComponentID1x2:   false,
				imageComponentID1s2x3: false,
				imageComponentID2x4:   false,
				imageComponentID2x5:   false,
			},
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
			// Therefore it should see all components.
			// (images are in cluster1 namespaceA and cluster2 namespaceB).
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:   true,
				imageComponentID1x2:   true,
				imageComponentID1s2x3: true,
				imageComponentID2x4:   true,
				imageComponentID2x5:   true,
			},
		},
	}

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

func (s *cveDataStoreSACTestSuite) TestSACImageComponentExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for Component1x1
	s.runImageTest("TestSACImageComponentExistsSingleScopeOnly", func(c componentTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageComponentStore.Exists(testCtx, imageComponentID1x1)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[imageComponentID1x1], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentExistsSharedComponent() {
	// Inject the fixture graph, and test exists for Component1s2x3
	s.runImageTest("TestSACImageComponentExistsSharedComponent", func(c componentTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageComponentStore.Exists(testCtx, imageComponentID1s2x3)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[imageComponentID1s2x3], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for Component1x1
	targetComponent := imageComponent1x1
	componentName := targetComponent.GetName()
	cvss := targetComponent.GetTopCvss()
	s.runImageTest("TestSACImageComponentGetSingleScopeOnly", func(c componentTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		imageComponent, found, err := s.imageComponentStore.Get(testCtx, imageComponentID1x1)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[imageComponentID1x1], found)
		if c.expectedComponentFound[imageComponentID1x1] {
			s.Require().NotNil(imageComponent)
			s.Equal(componentName, imageComponent.GetName())
			s.Equal(cvss, imageComponent.GetTopCvss())
		} else {
			s.Nil(imageComponent)
		}
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentGetSharedComponent() {
	// Inject the fixture graph, and test retrieval for Component1s2x3
	targetComponent := imageComponent1s2x3
	componentName := targetComponent.GetName()
	cvss := targetComponent.GetTopCvss()
	s.runImageTest("TestSACImageComponentGetSharedComponent", func(c componentTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		imageComponent, found, err := s.imageComponentStore.Get(testCtx, imageComponentID1s2x3)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[imageComponentID1s2x3], found)
		if c.expectedComponentFound[imageComponentID1s2x3] {
			s.Require().NotNil(imageComponent)
			s.Equal(componentName, imageComponent.GetName())
			s.Equal(cvss, imageComponent.GetTopCvss())
		} else {
			s.Nil(imageComponent)
		}
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentGetBatch() {
	componentIDs := []string{
		imageComponentID1x1,
		imageComponentID1x2,
		imageComponentID1s2x3,
		imageComponentID2x5,
	}
	s.runImageTest("TestSACImageComponentGetBatch", func(c componentTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		imageComponents, err := s.imageComponentStore.GetBatch(testCtx, componentIDs)
		s.NoError(err)
		expectedComponentIDs := make([]string, 0, len(componentIDs))
		for _, componentID := range componentIDs {
			if c.expectedComponentFound[componentID] {
				expectedComponentIDs = append(expectedComponentIDs, componentID)
			}
		}
		fetchedComponentIDs := make([]string, 0, len(imageComponents))
		for _, imageComponent := range imageComponents {
			fetchedComponentIDs = append(fetchedComponentIDs, imageComponent.GetId())
		}
		s.ElementsMatch(expectedComponentIDs, fetchedComponentIDs)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentCount() {
	s.T().Skip("Skipping Component count tests for now.")
	s.runImageTest("TestSACImageComponentCount", func(c componentTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		count, err := s.imageComponentStore.Count(testCtx, nil)
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

func (s *cveDataStoreSACTestSuite) TestSACImageComponentSearch() {
	s.runImageTest("", func(c componentTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageComponentStore.Search(testCtx, nil)
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

func (s *cveDataStoreSACTestSuite) TestSACImageComponentSearchImageComponents() {
	s.runImageTest("TestSACImageComponentSearchImageComponents", func(c componentTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageComponentStore.SearchImageComponents(testCtx, nil)
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

func (s *cveDataStoreSACTestSuite) TestSACImageComponentSearchRawImageComponents() {
	s.runImageTest("TestSACImageComponentSearchRawImageComponents", func(c componentTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageComponentStore.SearchRawImageComponents(testCtx, nil)
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

func (s *cveDataStoreSACTestSuite) runImageTest(testName string, testFunc func(c componentTestCase)) {
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)

	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)

	failed := false
	for _, c := range imageComponentTestCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			// s.T().Parallel()
			testFunc(c)
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
		log.Infof("%s failed, dumping DB content.", testName)
		nodeGraphBefore.Log()
	}
}
