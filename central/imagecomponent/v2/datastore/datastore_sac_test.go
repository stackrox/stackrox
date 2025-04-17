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
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/suite"
)

func TestImageComponentV2DataStoreSAC(t *testing.T) {
	suite.Run(t, new(componentV2DataStoreSACTestSuite))
}

type componentV2DataStoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore  graphDBTestUtils.TestGraphDataStore
	imageComponentStore DataStore

	imageTestContexts map[string]context.Context
}

func (s *componentV2DataStoreSACTestSuite) SetupSuite() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Setenv(features.FlattenCVEData.EnvVar(), "true")
	}

	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.imageComponentStore = GetTestPostgresDataStore(s.T(), pool)
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

func getImageComponentID(component *storage.EmbeddedImageScanComponent, imageID string) string {
	componentID, _ := scancomponent.ComponentIDV2(component, imageID)
	return componentID
}

func (s *componentV2DataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanImageToVulnerabilitiesGraph())
}

type componentTestCase struct {
	contextKey             string
	expectedComponentFound map[string]bool
}

var (
	imageComponent1x1       = fixtures.GetEmbeddedImageComponent1x1()
	imageComponent1x2       = fixtures.GetEmbeddedImageComponent1x2()
	imageComponent1s2x3     = fixtures.GetEmbeddedImageComponent1s2x3()
	imageComponent2x4       = fixtures.GetEmbeddedImageComponent2x4()
	imageComponent2x5       = fixtures.GetEmbeddedImageComponent2x5()
	imageComponentID1x1     = getImageComponentID(imageComponent1x1, fixtures.GetImageSherlockHolmes1().GetId())
	imageComponentID1x2     = getImageComponentID(imageComponent1x2, fixtures.GetImageSherlockHolmes1().GetId())
	imageComponentID1s2x3i1 = getImageComponentID(imageComponent1s2x3, fixtures.GetImageSherlockHolmes1().GetId())
	imageComponentID1s2x3i2 = getImageComponentID(imageComponent1s2x3, fixtures.GetImageDoctorJekyll2().GetId())
	imageComponentID2x4     = getImageComponentID(imageComponent2x4, fixtures.GetImageDoctorJekyll2().GetId())
	imageComponentID2x5     = getImageComponentID(imageComponent2x5, fixtures.GetImageDoctorJekyll2().GetId())

	imageComponentTestCases = []componentTestCase{
		{
			contextKey: sacTestUtils.UnrestrictedReadCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     true,
				imageComponentID1x2:     true,
				imageComponentID1s2x3i1: true,
				imageComponentID1s2x3i2: true,
				imageComponentID2x4:     true,
				imageComponentID2x5:     true,
			},
		},
		{
			contextKey: sacTestUtils.UnrestrictedReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     true,
				imageComponentID1x2:     true,
				imageComponentID1s2x3i1: true,
				imageComponentID1s2x3i2: true,
				imageComponentID2x4:     true,
				imageComponentID2x5:     true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     true,
				imageComponentID1x2:     true,
				imageComponentID1s2x3i1: true,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     true,
				imageComponentID1x2:     true,
				imageComponentID1s2x3i1: true,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     true,
				imageComponentID1x2:     true,
				imageComponentID1s2x3i1: true,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x2:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: true,
				imageComponentID2x4:     true,
				imageComponentID2x5:     true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: true,
				imageComponentID2x4:     true,
				imageComponentID2x5:     true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: true,
				imageComponentID2x4:     true,
				imageComponentID2x5:     true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     false,
				imageComponentID1x2:     false,
				imageComponentID1s2x3i1: false,
				imageComponentID1s2x3i2: false,
				imageComponentID2x4:     false,
				imageComponentID2x5:     false,
			},
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
			// Therefore, it should see all components.
			// (images are in cluster1 namespaceA and cluster2 namespaceB).
			expectedComponentFound: map[string]bool{
				imageComponentID1x1:     true,
				imageComponentID1x2:     true,
				imageComponentID1s2x3i1: true,
				imageComponentID1s2x3i2: true,
				imageComponentID2x4:     true,
				imageComponentID2x5:     true,
			},
		},
	}
)

func (s *componentV2DataStoreSACTestSuite) TestSACImageComponentExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for Component1x1
	s.runImageTest("TestSACImageComponentExistsSingleScopeOnly", func(c componentTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageComponentStore.Exists(testCtx, imageComponentID1x1)
		s.NoError(err)
		s.Equal(c.expectedComponentFound[imageComponentID1x1], exists)
	})
}

func (s *componentV2DataStoreSACTestSuite) TestSACImageComponentGetSingleScopeOnly() {
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

func (s *componentV2DataStoreSACTestSuite) TestSACImageComponentGetBatch() {
	componentIDs := []string{
		imageComponentID1x1,
		imageComponentID1x2,
		imageComponentID1s2x3i1,
		imageComponentID1s2x3i2,
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

func (s *componentV2DataStoreSACTestSuite) TestSACImageComponentCount() {
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

func (s *componentV2DataStoreSACTestSuite) TestSACImageComponentSearch() {
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

func (s *componentV2DataStoreSACTestSuite) TestSACImageComponentSearchImageComponents() {
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

func (s *componentV2DataStoreSACTestSuite) TestSACImageComponentSearchRawImageComponents() {
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

func (s *componentV2DataStoreSACTestSuite) runImageTest(testName string, testFunc func(c componentTestCase)) {
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
