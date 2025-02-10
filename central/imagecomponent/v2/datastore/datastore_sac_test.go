//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/suite"
)

func TestImageComponentDataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveDataStoreSACTestSuite))
}

type cveDataStoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore  graphDBTestUtils.TestGraphDataStore
	imageComponentStore DataStore

	imageTestContexts map[string]context.Context
}

func (s *cveDataStoreSACTestSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.imageComponentStore = GetTestPostgresDataStore(s.T(), pool)
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

func (s *cveDataStoreSACTestSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

func getImageComponentID(component *storage.EmbeddedImageScanComponent, imageID string) string {
	return scancomponent.ComponentIDV2(component.GetName(), component.GetVersion(), component.GetArchitecture(), imageID)
}

func (s *cveDataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanImageToVulnerabilitiesGraph())
}

type componentTestCase struct {
	contextKey             string
	expectedComponentFound map[string]bool
}

var (
	image1ID              = fixtures.GetImageSherlockHolmes1().GetId()
	image2ID              = fixtures.GetImageDoctorJekyll2().GetId()
	imageComponent1x1     = fixtures.GetEmbeddedImageComponent1x1()
	imageComponent1x2     = fixtures.GetEmbeddedImageComponent1x2()
	imageComponent1s2x3   = fixtures.GetEmbeddedImageComponent1s2x3()
	imageComponent2x4     = fixtures.GetEmbeddedImageComponent2x4()
	imageComponent2x5     = fixtures.GetEmbeddedImageComponent2x5()
	imageComponentID1x1   = getImageComponentID(imageComponent1x1, image1ID)
	imageComponentID1x2   = getImageComponentID(imageComponent1x2, image1ID)
	imageComponentID1s2x3 = getImageComponentID(imageComponent1s2x3, image1ID)
	imageComponentID2x4   = getImageComponentID(imageComponent2x4, image2ID)
	imageComponentID2x5   = getImageComponentID(imageComponent2x5, image2ID)

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
			// Therefore, it should see all components.
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
)

func (s *cveDataStoreSACTestSuite) TestSACImageComponentExistsSingleScopeOnly() {
	s.T().Skip("Skipping Component tests for now until image store updates complete")
	// Inject the fixture graph, and test exists for Component1x1
	s.runImageTest("TestSACImageComponentExistsSingleScopeOnly", func(c componentTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageComponentStore.Exists(testCtx, imageComponentID1x1)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[imageComponentID1x1], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentExistsSharedComponent() {
	s.T().Skip("Skipping Component tests for now until image store updates complete")
	// Inject the fixture graph, and test exists for Component1s2x3
	s.runImageTest("TestSACImageComponentExistsSharedComponent", func(c componentTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageComponentStore.Exists(testCtx, imageComponentID1s2x3)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[imageComponentID1s2x3], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageComponentGetSingleScopeOnly() {
	s.T().Skip("Skipping Component tests for now until image store updates complete")

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
	s.T().Skip("Skipping Component tests for now until image store updates complete")

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
	s.T().Skip("Skipping Component tests for now until image store updates complete")

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
	s.T().Skip("Skipping Component tests for now until image store updates complete")

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
	s.T().Skip("Skipping Component tests for now until image store updates complete")

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
	s.T().Skip("Skipping Component tests for now until image store updates complete")

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
