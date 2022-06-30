package datastoretest

import (
	"context"
	"testing"

	dackboxTestUtils "github.com/stackrox/rox/central/dackbox/testutils"
	imageComponentDataStore "github.com/stackrox/rox/central/imagecomponent/datastore"
	nodeComponentDataStore "github.com/stackrox/rox/central/nodecomponent/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/suite"
)

func TestCVEDataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveDataStoreSACTestSuite))
}

type cveDataStoreSACTestSuite struct {
	suite.Suite

	dackboxTestStore    dackboxTestUtils.DackboxTestDataStore
	imageComponentStore imageComponentDataStore.DataStore
	nodeComponentStore  nodeComponentDataStore.DataStore

	nodeTestContexts  map[string]context.Context
	imageTestContexts map[string]context.Context
}

func (s *cveDataStoreSACTestSuite) SetupSuite() {
	var err error
	s.dackboxTestStore, err = dackboxTestUtils.NewDackboxTestDataStore(s.T())
	s.Require().NoError(err)
	if features.PostgresDatastore.Enabled() {
		pool := s.dackboxTestStore.GetPostgresPool()
		s.imageComponentStore, err = imageComponentDataStore.GetTestPostgresDataStore(s.T(), pool)
		s.Require().NoError(err)
		s.nodeComponentStore, err = nodeComponentDataStore.GetTestPostgresDataStore(s.T(), pool)
		s.Require().NoError(err)
	} else {
		dacky := s.dackboxTestStore.GetDackbox()
		keyFence := s.dackboxTestStore.GetKeyFence()
		rocksEngine := s.dackboxTestStore.GetRocksEngine()
		bleveIndex := s.dackboxTestStore.GetBleveIndex()
		s.imageComponentStore, err = imageComponentDataStore.GetTestRocksBleveDataStore(s.T(), rocksEngine, bleveIndex, dacky,
			keyFence)
		s.Require().NoError(err)
		s.nodeComponentStore = &nodeComponentFromImageComponentDataStore{
			imageComponentStore: s.imageComponentStore,
		}
	}
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
	s.nodeTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
}

func (s *cveDataStoreSACTestSuite) TearDownSuite() {
	s.Require().NoError(s.dackboxTestStore.Cleanup(s.T()))
}

func getImageComponentId(component *storage.EmbeddedImageScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func getNodeComponentId(component *storage.EmbeddedNodeScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func (s *cveDataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.dackboxTestStore.CleanImageToVulnerabilitiesGraph())
}

func (s *cveDataStoreSACTestSuite) cleanNodeToVulnerabilitiesGraph() {
	s.Require().NoError(s.dackboxTestStore.CleanNodeToVulnerabilitiesGraph())
}

type componentTestCase struct {
	contextKey             string
	expectedComponentFound map[string]bool
}

var (
	image_1_OS              = fixtures.GetImageSherlockHolmes_1().GetScan().GetOperatingSystem()
	image_2_OS              = fixtures.GetImageDoctorJekyll_2().GetScan().GetOperatingSystem()
	image_component_1_1     = fixtures.GetEmbeddedImageComponent_1_1()
	image_component_1_2     = fixtures.GetEmbeddedImageComponent_1_2()
	image_component_1s2_3   = fixtures.GetEmbeddedImageComponent_1s2_3()
	image_component_2_4     = fixtures.GetEmbeddedImageComponent_2_4()
	image_component_2_5     = fixtures.GetEmbeddedImageComponent_2_5()
	imageComponent_1_1_ID   = getImageComponentId(image_component_1_1, image_1_OS)
	imageComponent_1_2_ID   = getImageComponentId(image_component_1_2, image_1_OS)
	imageComponent_1s2_3_ID = getImageComponentId(image_component_1s2_3, image_1_OS)
	imageComponent_2_4_ID   = getImageComponentId(image_component_2_4, image_2_OS)
	imageComponent_2_5_ID   = getImageComponentId(image_component_2_5, image_2_OS)

	imageComponentTestCases = []componentTestCase{
		{
			contextKey: sacTestUtils.UnrestrictedReadCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   true,
				imageComponent_1_2_ID:   true,
				imageComponent_1s2_3_ID: true,
				imageComponent_2_4_ID:   true,
				imageComponent_2_5_ID:   true,
			},
		},
		{
			contextKey: sacTestUtils.UnrestrictedReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   true,
				imageComponent_1_2_ID:   true,
				imageComponent_1s2_3_ID: true,
				imageComponent_2_4_ID:   true,
				imageComponent_2_5_ID:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   true,
				imageComponent_1_2_ID:   true,
				imageComponent_1s2_3_ID: true,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   true,
				imageComponent_1_2_ID:   true,
				imageComponent_1s2_3_ID: true,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: false,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   true,
				imageComponent_1_2_ID:   true,
				imageComponent_1s2_3_ID: true,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: false,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: true,
				imageComponent_2_4_ID:   true,
				imageComponent_2_5_ID:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: false,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: true,
				imageComponent_2_4_ID:   true,
				imageComponent_2_5_ID:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: false,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: true,
				imageComponent_2_4_ID:   true,
				imageComponent_2_5_ID:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: false,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: false,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: false,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponent_1_1_ID:   false,
				imageComponent_1_2_ID:   false,
				imageComponent_1s2_3_ID: false,
				imageComponent_2_4_ID:   false,
				imageComponent_2_5_ID:   false,
			},
		},
		/*
			{
				contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
				// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
				// Therefore it should see all components.
				// (images are in cluster1 namespaceA and cluster2 namespaceB).
				expectedComponentFound: map[string]bool{
					imageComponent_1_1_ID:   true,
					imageComponent_1_2_ID:   true,
					imageComponent_1s2_3_ID: true,
					imageComponent_2_4_ID:   true,
					imageComponent_2_5_ID:   true,
				},
			},
		*/
	}

	node_1_OS              = fixtures.GetScopedNode_1("dummyNodeID", testconsts.Cluster1).GetScan().GetOperatingSystem()
	node_2_OS              = fixtures.GetScopedNode_1("dummyNodeID", testconsts.Cluster2).GetScan().GetOperatingSystem()
	node_component_1_1     = fixtures.GetEmbeddedNodeComponent_1_1()
	node_component_1_2     = fixtures.GetEmbeddedNodeComponent_1_2()
	node_component_1s2_3   = fixtures.GetEmbeddedNodeComponent_1s2_3()
	node_component_2_4     = fixtures.GetEmbeddedNodeComponent_2_4()
	node_component_2_5     = fixtures.GetEmbeddedNodeComponent_2_5()
	nodeComponent_1_1_ID   = getNodeComponentId(node_component_1_1, node_1_OS)
	nodeComponent_1_2_ID   = getNodeComponentId(node_component_1_2, node_1_OS)
	nodeComponent_1s2_3_ID = getNodeComponentId(node_component_1s2_3, node_1_OS)
	nodeComponent_2_4_ID   = getNodeComponentId(node_component_2_4, node_2_OS)
	nodeComponent_2_5_ID   = getNodeComponentId(node_component_2_5, node_2_OS)

	nodeComponentTestCases = []componentTestCase{
		{
			contextKey: sacTestUtils.UnrestrictedReadCtx,
			expectedComponentFound: map[string]bool{
				nodeComponent_1_1_ID:   true,
				nodeComponent_1_2_ID:   true,
				nodeComponent_1s2_3_ID: true,
				nodeComponent_2_4_ID:   true,
				nodeComponent_2_5_ID:   true,
			},
		},
		{
			contextKey: sacTestUtils.UnrestrictedReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponent_1_1_ID:   true,
				nodeComponent_1_2_ID:   true,
				nodeComponent_1s2_3_ID: true,
				nodeComponent_2_4_ID:   true,
				nodeComponent_2_5_ID:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponent_1_1_ID:   true,
				nodeComponent_1_2_ID:   true,
				nodeComponent_1s2_3_ID: true,
				nodeComponent_2_4_ID:   false,
				nodeComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			// Partial cluster scope is too narrow for allowfixedscope at cluster level.
			expectedComponentFound: map[string]bool{
				nodeComponent_1_1_ID:   false,
				nodeComponent_1_2_ID:   false,
				nodeComponent_1s2_3_ID: false,
				nodeComponent_2_4_ID:   false,
				nodeComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponent_1_1_ID:   false,
				nodeComponent_1_2_ID:   false,
				nodeComponent_1s2_3_ID: true,
				nodeComponent_2_4_ID:   true,
				nodeComponent_2_5_ID:   true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			// Partial cluster scope is too narrow for allowfixedscope at cluster level.
			expectedComponentFound: map[string]bool{
				nodeComponent_1_1_ID:   false,
				nodeComponent_1_2_ID:   false,
				nodeComponent_1s2_3_ID: false,
				nodeComponent_2_4_ID:   false,
				nodeComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				nodeComponent_1_1_ID:   false,
				nodeComponent_1_2_ID:   false,
				nodeComponent_1s2_3_ID: false,
				nodeComponent_2_4_ID:   false,
				nodeComponent_2_5_ID:   false,
			},
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
			// Therefore it should see only cluster2 vulnerabilities (and shared ones).
			expectedComponentFound: map[string]bool{
				nodeComponent_1_1_ID:   false,
				nodeComponent_1_2_ID:   false,
				nodeComponent_1s2_3_ID: true,
				nodeComponent_2_4_ID:   true,
				nodeComponent_2_5_ID:   true,
			},
		},
	}
)

func (s *cveDataStoreSACTestSuite) TestSACImageComponentExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for Component_1_1
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	for _, c := range imageComponentTestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			exists, err := s.imageComponentStore.Exists(testCtx, imageComponent_1_1_ID)
			s.NoError(err)
			s.Equal(c.expectedComponentFound[imageComponent_1_1_ID], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentExistsSharedComponent() {
	// Inject the fixture graph, and test exists for Component_1s2_3
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	for _, c := range imageComponentTestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			exists, err := s.imageComponentStore.Exists(testCtx, imageComponent_1s2_3_ID)
			s.NoError(err)
			s.Equal(c.expectedComponentFound[imageComponent_1s2_3_ID], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for Component_1_1
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	targetComponent := image_component_1_1
	componentName := targetComponent.GetName()
	cvss := targetComponent.GetTopCvss()
	for _, c := range imageComponentTestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			imageComponent, found, err := s.imageComponentStore.Get(testCtx, imageComponent_1_1_ID)
			s.NoError(err)
			s.Equal(c.expectedComponentFound[imageComponent_1_1_ID], found)
			if c.expectedComponentFound[imageComponent_1_1_ID] {
				s.Require().NotNil(imageComponent)
				s.Equal(componentName, imageComponent.GetName())
				s.Equal(cvss, imageComponent.GetTopCvss())
			} else {
				s.Nil(imageComponent)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentGetSharedComponent() {
	// Inject the fixture graph, and test retrieval for Component_1s2_3
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	targetComponent := image_component_1s2_3
	componentName := targetComponent.GetName()
	cvss := targetComponent.GetTopCvss()
	for _, c := range imageComponentTestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			imageComponent, found, err := s.imageComponentStore.Get(testCtx, imageComponent_1s2_3_ID)
			s.NoError(err)
			s.Equal(c.expectedComponentFound[imageComponent_1s2_3_ID], found)
			if c.expectedComponentFound[imageComponent_1s2_3_ID] {
				s.Require().NotNil(imageComponent)
				s.Equal(componentName, imageComponent.GetName())
				s.Equal(cvss, imageComponent.GetTopCvss())
			} else {
				s.Nil(imageComponent)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentGetBatch() {
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	componentIDs := []string{
		imageComponent_1_1_ID,
		imageComponent_1_2_ID,
		imageComponent_1s2_3_ID,
		imageComponent_2_5_ID,
	}
	for _, c := range imageComponentTestCases {
		s.Run(c.contextKey, func() {
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
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentCount() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentSearch() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentSearchImageComponents() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentSearchRawImageComponents() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for Component_1_1
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	for _, c := range nodeComponentTestCases {
		s.Run(c.contextKey, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			exists, err := s.nodeComponentStore.Exists(testCtx, nodeComponent_1_1_ID)
			s.NoError(err)
			s.Equal(c.expectedComponentFound[nodeComponent_1_1_ID], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentExistsSharedComponent() {
	// Inject the fixture graph, and test exists for Component_1s2_3
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	for _, c := range nodeComponentTestCases {
		s.Run(c.contextKey, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			exists, err := s.nodeComponentStore.Exists(testCtx, nodeComponent_1s2_3_ID)
			s.NoError(err)
			s.Equal(c.expectedComponentFound[nodeComponent_1s2_3_ID], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for Component_1_1
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	targetComponent := node_component_1_1
	componentName := targetComponent.GetName()
	cvss := targetComponent.GetTopCvss()
	for _, c := range nodeComponentTestCases {
		s.Run(c.contextKey, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeComponent, found, err := s.nodeComponentStore.Get(testCtx, nodeComponent_1_1_ID)
			s.NoError(err)
			s.Equal(c.expectedComponentFound[nodeComponent_1_1_ID], found)
			if c.expectedComponentFound[nodeComponent_1_1_ID] {
				s.NotNil(nodeComponent)
				s.Equal(componentName, nodeComponent.GetName())
				s.Equal(cvss, nodeComponent.GetTopCvss())
			} else {
				s.Nil(nodeComponent)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentGetSharedComponent() {
	// Inject the fixture graph, and test retrieval for Component_1s2_3
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	targetComponent := node_component_1s2_3
	componentName := targetComponent.GetName()
	cvss := targetComponent.GetTopCvss()
	for _, c := range nodeComponentTestCases {
		s.Run(c.contextKey, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeComponent, found, err := s.nodeComponentStore.Get(testCtx, nodeComponent_1s2_3_ID)
			s.NoError(err)
			s.Equal(c.expectedComponentFound[nodeComponent_1s2_3_ID], found)
			if c.expectedComponentFound[nodeComponent_1s2_3_ID] {
				s.NotNil(nodeComponent)
				s.Equal(componentName, nodeComponent.GetName())
				s.Equal(cvss, nodeComponent.GetTopCvss())
			} else {
				s.Nil(nodeComponent)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentGetBatch() {
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	componentIDs := []string{
		nodeComponent_1_1_ID,
		nodeComponent_1_2_ID,
		nodeComponent_1s2_3_ID,
		nodeComponent_2_5_ID,
	}
	for _, c := range nodeComponentTestCases {
		s.Run(c.contextKey, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
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
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentCount() {
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	for _, c := range nodeComponentTestCases {
		s.Run(c.contextKey, func() {

			s.T().Skip("Skipping CVE count tests for now.")

			testCtx := s.nodeTestContexts[c.contextKey]
			count, err := s.nodeComponentStore.Count(testCtx, nil)
			s.NoError(err)
			expectedCount := 0
			for _, visible := range c.expectedComponentFound {
				if visible {
					expectedCount += 1
				}
			}
			s.Equal(expectedCount, count)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentSearch() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentSearchNodeComponents() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACNodeComponentSearchRawNodeComponents() {
	s.T().Skip("Not implemented yet.")
}
