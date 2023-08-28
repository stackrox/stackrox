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

var (
	imageScanOperatingSystem = "crime-stories"

	log = logging.LoggerForModule()

	dontWaitForIndexing = false
	waitForIndexing     = true
)

func TestImageComponentEdgeDatastoreSAC(t *testing.T) {
	suite.Run(t, new(imageComponentEdgeDatastoreSACTestSuite))
}

type imageComponentEdgeDatastoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore graphDBTestUtils.TestGraphDataStore
	datastore          DataStore

	testContexts map[string]context.Context
}

func (s *imageComponentEdgeDatastoreSACTestSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.datastore, err = GetTestPostgresDataStore(s.T(), pool)
	s.Require().NoError(err)
	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

func (s *imageComponentEdgeDatastoreSACTestSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

func (s *imageComponentEdgeDatastoreSACTestSuite) cleanImageToVulnerabilityGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanImageToVulnerabilitiesGraph())
}

func getComponentID(component *storage.EmbeddedImageScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func getEdgeID(image *storage.Image, component *storage.EmbeddedImageScanComponent, os string) string {
	imageID := image.GetId()
	componentID := getComponentID(component, os)
	return pgSearch.IDFromPks([]string{imageID, componentID})
}

type edgeTestCase struct {
	contextKey        string
	expectedEdgeFound map[string]bool
}

var (
	img1cmp1edge = getEdgeID(fixtures.GetImageSherlockHolmes1(), fixtures.GetEmbeddedImageComponent1x1(), imageScanOperatingSystem)
	img1cmp2edge = getEdgeID(fixtures.GetImageSherlockHolmes1(), fixtures.GetEmbeddedImageComponent1x2(), imageScanOperatingSystem)
	img1cmp3edge = getEdgeID(fixtures.GetImageSherlockHolmes1(), fixtures.GetEmbeddedImageComponent1s2x3(), imageScanOperatingSystem)
	img2cmp3edge = getEdgeID(fixtures.GetImageDoctorJekyll2(), fixtures.GetEmbeddedImageComponent1s2x3(), imageScanOperatingSystem)
	img2cmp4edge = getEdgeID(fixtures.GetImageDoctorJekyll2(), fixtures.GetEmbeddedImageComponent2x4(), imageScanOperatingSystem)
	img2cmp5edge = getEdgeID(fixtures.GetImageDoctorJekyll2(), fixtures.GetEmbeddedImageComponent2x5(), imageScanOperatingSystem)

	fullAccessMap = map[string]bool{
		img1cmp1edge: true,
		img1cmp2edge: true,
		img1cmp3edge: true,
		img2cmp3edge: true,
		img2cmp4edge: true,
		img2cmp5edge: true,
	}
	cluster1WithNamespaceAAccessMap = map[string]bool{
		img1cmp1edge: true,
		img1cmp2edge: true,
		img1cmp3edge: true,
		img2cmp3edge: false,
		img2cmp4edge: false,
		img2cmp5edge: false,
	}
	cluster2WithNamespaceBAccessMap = map[string]bool{
		img1cmp1edge: false,
		img1cmp2edge: false,
		img1cmp3edge: false,
		img2cmp3edge: true,
		img2cmp4edge: true,
		img2cmp5edge: true,
	}
	noAccessMap = map[string]bool{
		img1cmp1edge: false,
		img1cmp2edge: false,
		img1cmp3edge: false,
		img2cmp3edge: false,
		img2cmp4edge: false,
		img2cmp5edge: false,
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
			expectedEdgeFound: cluster1WithNamespaceAAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedEdgeFound: cluster1WithNamespaceAAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedEdgeFound: cluster1WithNamespaceAAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2ReadWriteCtx,
			expectedEdgeFound: cluster2WithNamespaceBAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedEdgeFound: cluster2WithNamespaceBAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespacesABReadWriteCtx,
			expectedEdgeFound: cluster2WithNamespaceBAccessMap,
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
			// Has access to Cluster1 + NamespaceA as well as full access to Cluster2 (including NamespaceB).
			expectedEdgeFound: fullAccessMap,
		},
	}
)

func (s *imageComponentEdgeDatastoreSACTestSuite) TestExistsEdge() {
	// Inject the fixture graph and test for image1 to component1 edge
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilityGraph()
	s.Require().NoError(err)
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false

	targetEdgeID := img1cmp1edge
	for _, c := range testCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			ctx := s.testContexts[c.contextKey]
			exists, err := s.datastore.Exists(ctx, targetEdgeID)
			s.NoError(err)
			s.Equal(c.expectedEdgeFound[targetEdgeID], exists)
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Info("TestExistsEdge failed, dumping DB content.")
		imageGraphBefore.Log()
	}
}

func (s *imageComponentEdgeDatastoreSACTestSuite) TestGetEdge() {
	// Inject the fixtures graph and fetch the image1 to component1 edge
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilityGraph()
	s.Require().NoError(err)
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false

	targetEdgeID := img1cmp1edge
	expectedSrcID := fixtures.GetImageSherlockHolmes1().GetId()
	expectedDstID := getComponentID(fixtures.GetEmbeddedImageComponent1x1(), imageScanOperatingSystem)
	for _, c := range testCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			ctx := s.testContexts[c.contextKey]
			fetched, found, err := s.datastore.Get(ctx, targetEdgeID)
			s.NoError(err)
			if c.expectedEdgeFound[targetEdgeID] {
				s.True(found)
				s.Require().NotNil(fetched)
				s.Equal(expectedSrcID, fetched.GetImageId())
				s.Equal(expectedDstID, fetched.GetImageComponentId())
			} else {
				s.False(found)
				s.Nil(fetched)
			}
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Info("TestGetEdge failed, dumping DB content.")
		imageGraphBefore.Log()
	}
}

func (s *imageComponentEdgeDatastoreSACTestSuite) TestGetBatch() {
	// Inject the fixtures graph and fetch the image1 to component1 and image2 to component 4 edges
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilityGraph()
	s.Require().NoError(err)
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false

	targetEdge1ID := img1cmp1edge
	expectedSrc1ID := fixtures.GetImageSherlockHolmes1().GetId()
	expectedDst1ID := getComponentID(fixtures.GetEmbeddedImageComponent1x1(), imageScanOperatingSystem)
	targetEdge2ID := img2cmp4edge
	expectedSrc2ID := fixtures.GetImageDoctorJekyll2().GetId()
	expectedDst2ID := getComponentID(fixtures.GetEmbeddedImageComponent2x4(), imageScanOperatingSystem)
	toFetch := []string{targetEdge1ID, targetEdge2ID}
	for _, c := range testCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			ctx := s.testContexts[c.contextKey]
			fetched, err := s.datastore.GetBatch(ctx, toFetch)
			s.NoError(err)
			expectedFetchedSize := 0
			if c.expectedEdgeFound[targetEdge1ID] {
				expectedFetchedSize++
			}
			if c.expectedEdgeFound[targetEdge2ID] {
				expectedFetchedSize++
			}
			fetchedMatches := 0
			s.Equal(expectedFetchedSize, len(fetched))
			for _, edge := range fetched {
				if edge.GetId() == targetEdge1ID {
					fetchedMatches++
					s.Equal(expectedSrc1ID, edge.GetImageId())
					s.Equal(expectedDst1ID, edge.GetImageComponentId())
				}
				if edge.GetId() == targetEdge2ID {
					fetchedMatches++
					s.Equal(expectedSrc2ID, edge.GetImageId())
					s.Equal(expectedDst2ID, edge.GetImageComponentId())
				}
			}
			s.Equal(expectedFetchedSize, fetchedMatches)
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Info("TestGetBatch failed, dumping DB content.")
		imageGraphBefore.Log()
	}
}

func (s *imageComponentEdgeDatastoreSACTestSuite) TestCount() {
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilityGraph()
	s.Require().NoError(err)
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false

	for _, c := range testCases {
		caseSucceeded := s.Run(c.contextKey, func() {
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
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Info("TestCount failed, dumping DB content.")
		imageGraphBefore.Log()
	}
}

func (s *imageComponentEdgeDatastoreSACTestSuite) TestSearch() {
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilityGraph()
	s.Require().NoError(err)
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false

	for _, c := range testCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			ctx := s.testContexts[c.contextKey]
			expectedCount := 0
			for _, visible := range c.expectedEdgeFound {
				if visible {
					expectedCount++
				}
			}
			results, err := s.datastore.Search(ctx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(expectedCount, len(results))
			for _, r := range results {
				s.True(c.expectedEdgeFound[r.ID])
			}
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Info("TestSearch failed, dumping DB content.")
		imageGraphBefore.Log()
	}
}

func (s *imageComponentEdgeDatastoreSACTestSuite) TestSearchEdges() {
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilityGraph()
	s.Require().NoError(err)
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false

	for _, c := range testCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			ctx := s.testContexts[c.contextKey]
			expectedCount := 0
			for _, visible := range c.expectedEdgeFound {
				if visible {
					expectedCount++
				}
			}
			results, err := s.datastore.SearchEdges(ctx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(expectedCount, len(results))
			for _, r := range results {
				s.True(c.expectedEdgeFound[r.GetId()])
			}
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Info("TestSearchEdges failed, dumping DB content.")
		imageGraphBefore.Log()
	}

}

func (s *imageComponentEdgeDatastoreSACTestSuite) TestSearchRawEdges() {
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilityGraph()
	s.Require().NoError(err)
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)
	failed := false

	for _, c := range testCases {
		caseSucceeded := s.Run(c.contextKey, func() {
			ctx := s.testContexts[c.contextKey]
			expectedCount := 0
			for _, visible := range c.expectedEdgeFound {
				if visible {
					expectedCount++
				}
			}
			results, err := s.datastore.SearchRawEdges(ctx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(expectedCount, len(results))
			for _, r := range results {
				s.True(c.expectedEdgeFound[r.GetId()])
			}
		})
		if !caseSucceeded {
			failed = true
		}
	}
	if failed {
		log.Info("TestSearchRawEdges failed, dumping DB content.")
		imageGraphBefore.Log()
	}

}
