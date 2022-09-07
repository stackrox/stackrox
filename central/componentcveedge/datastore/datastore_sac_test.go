package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cve/converter/utils"
	dackboxTestUtils "github.com/stackrox/rox/central/dackbox/testutils"
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
	imageScanOperatingSystem = "crime-stories"

	dontWaitForIndexing = false
	waitForIndexing     = true
)

func TestImageComponentCVEEdgeDatastoreSAC(t *testing.T) {
	suite.Run(t, new(imageComponentCVEEdgeDatastoreSACTestSuite))
}

type imageComponentCVEEdgeDatastoreSACTestSuite struct {
	suite.Suite

	dackboxTestStore dackboxTestUtils.DackboxTestDataStore
	datastore        DataStore

	testContexts map[string]context.Context
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) SetupSuite() {
	var err error
	s.dackboxTestStore, err = dackboxTestUtils.NewDackboxTestDataStore(s.T())
	s.Require().NoError(err)
	if features.PostgresDatastore.Enabled() {
		pool := s.dackboxTestStore.GetPostgresPool()
		s.datastore, err = GetTestPostgresDataStore(s.T(), pool)
		s.Require().NoError(err)
	} else {
		rocksengine := s.dackboxTestStore.GetRocksEngine()
		bleveIndex := s.dackboxTestStore.GetBleveIndex()
		dacky := s.dackboxTestStore.GetDackbox()
		s.datastore, err = GetTestRocksBleveDataStore(s.T(), rocksengine, bleveIndex, dacky)
		s.Require().NoError(err)
	}
	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TearDownSuite() {
	s.Require().NoError(s.dackboxTestStore.Cleanup(s.T()))
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) cleanImageToVulnerabilitiesGraph(waitForIndexing bool) {
	s.Require().NoError(s.dackboxTestStore.CleanImageToVulnerabilitiesGraph(waitForIndexing))
}

func getComponentID(component *storage.EmbeddedImageScanComponent, os string) string {
	return scancomponent.ComponentID(component.GetName(), component.GetVersion(), os)
}

func getCveID(vulnerability *storage.EmbeddedVulnerability, os string) string {
	return utils.EmbeddedCVEToProtoCVE(os, vulnerability).GetId()
}

func getEdgeID(component *storage.EmbeddedImageScanComponent, vulnerability *storage.EmbeddedVulnerability, os string) string {
	componentID := getComponentID(component, os)
	convertedCVEID := getCveID(vulnerability, os)
	if features.PostgresDatastore.Enabled() {
		return postgres.IDFromPks([]string{componentID, convertedCVEID})
	}
	return edges.EdgeID{ParentID: componentID, ChildID: convertedCVEID}.ToString()
}

type edgeTestCase struct {
	contextKey        string
	expectedEdgeFound map[string]bool
}

var (
	cmp1cve1edge = getEdgeID(fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetEmbeddedImageCVE1234x0001(), imageScanOperatingSystem)
	cmp1cve2edge = getEdgeID(fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetEmbeddedImageCVE4567x0002(), imageScanOperatingSystem)
	cmp2cve3edge = getEdgeID(fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetEmbeddedImageCVE1234x0003(), imageScanOperatingSystem)
	cmp3cve4edge = getEdgeID(fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetEmbeddedImageCVE3456x0004(), imageScanOperatingSystem)
	cmp3cve5edge = getEdgeID(fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetEmbeddedImageCVE3456x0005(), imageScanOperatingSystem)
	cmp5cve2edge = getEdgeID(fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetEmbeddedImageCVE4567x0002(), imageScanOperatingSystem)
	cmp5cve6edge = getEdgeID(fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetEmbeddedImageCVE2345x0006(), imageScanOperatingSystem)
	cmp5cve7edge = getEdgeID(fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetEmbeddedImageCVE2345x0007(), imageScanOperatingSystem)

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

	cluster1WithNamespaceAMap = map[string]bool{
		cmp1cve1edge: true,
		cmp1cve2edge: true,
		cmp2cve3edge: true,
		cmp3cve4edge: true,
		cmp3cve5edge: true,
		cmp5cve2edge: false,
		cmp5cve6edge: false,
		cmp5cve7edge: false,
	}

	cluster2WithNamespaceBMap = map[string]bool{
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
			expectedEdgeFound: cluster1WithNamespaceAMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedEdgeFound: cluster1WithNamespaceAMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedEdgeFound: cluster1WithNamespaceAMap,
		},
		{
			contextKey:        sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2ReadWriteCtx,
			expectedEdgeFound: cluster2WithNamespaceBMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedEdgeFound: noAccessMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedEdgeFound: cluster2WithNamespaceBMap,
		},
		{
			contextKey:        sacTestUtils.Cluster2NamespacesABReadWriteCtx,
			expectedEdgeFound: cluster2WithNamespaceBMap,
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

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSingleComponent() {
	// Inject the fixture graph, and test exists for Component1 to CVE-1234-0001 edge
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeId := cmp1cve1edge
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			exists, err := s.datastore.Exists(testCtx, targetEdgeId)
			s.NoError(err)
			s.Equal(c.expectedEdgeFound[targetEdgeId], exists)
		})
	}
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSingleComponentToSharedCVE() {
	// Inject the fixture graph, and test exists for Component1 to CVE-4567-0002 edge
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeId := cmp1cve2edge
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			exists, err := s.datastore.Exists(testCtx, targetEdgeId)
			s.NoError(err)
			s.Equal(c.expectedEdgeFound[targetEdgeId], exists)
		})
	}
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestExistsEdgeFromSharedComponent() {
	// Inject the fixture graph, and test exists for Component3 to CVE-3456-0004 edge
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeId := cmp3cve4edge
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			exists, err := s.datastore.Exists(testCtx, targetEdgeId)
			s.NoError(err)
			s.Equal(c.expectedEdgeFound[targetEdgeId], exists)
		})
	}
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSingleComponent() {
	// Inject the fixture graph, and test read for Component1 to CVE-1234-0001 edge
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp1cve1edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedImageComponent1x1(), imageScanOperatingSystem)
	expectedTargetID := getCveID(fixtures.GetEmbeddedImageCVE1234x0001(), imageScanOperatingSystem)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
			s.NoError(err)
			if c.expectedEdgeFound[targetEdgeID] {
				s.True(exists)
				s.NotNil(edge)
				s.Equal(expectedSrcID, edge.GetImageComponentId())
				s.Equal(expectedTargetID, edge.GetImageCveId())
			} else {
				s.False(exists)
				s.Nil(edge)
			}
		})
	}
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSingleComponentToSharedCVE() {
	// Inject the fixture graph, and test read for Component1 to CVE-4567-0002 edge
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp1cve2edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedImageComponent1x1(), imageScanOperatingSystem)
	expectedTargetID := getCveID(fixtures.GetEmbeddedImageCVE4567x0002(), imageScanOperatingSystem)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
			s.NoError(err)
			if c.expectedEdgeFound[targetEdgeID] {
				s.True(exists)
				s.NotNil(edge)
				s.Equal(expectedSrcID, edge.GetImageComponentId())
				s.Equal(expectedTargetID, edge.GetImageCveId())
			} else {
				s.False(exists)
				s.Nil(edge)
			}
		})
	}
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestGetEdgeFromSharedComponent() {
	// Inject the fixture graph, and test read for Component3 to CVE-3456-0004 edge
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(dontWaitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(dontWaitForIndexing)
	s.Require().NoError(err)
	targetEdgeID := cmp3cve4edge
	expectedSrcID := getComponentID(fixtures.GetEmbeddedImageComponent1s2x3(), imageScanOperatingSystem)
	expectedTargetID := getCveID(fixtures.GetEmbeddedImageCVE3456x0004(), imageScanOperatingSystem)
	for _, c := range testCases {
		s.Run(c.contextKey, func() {
			testCtx := s.testContexts[c.contextKey]
			edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
			s.NoError(err)
			if c.expectedEdgeFound[targetEdgeID] {
				s.True(exists)
				s.NotNil(edge)
				s.Equal(expectedSrcID, edge.GetImageComponentId())
				s.Equal(expectedTargetID, edge.GetImageCveId())
			} else {
				s.False(exists)
				s.Nil(edge)
			}
		})
	}
}

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestCount() {
	// Inject the fixture graph, and test data filtering on count operations
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(waitForIndexing)
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

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestSearch() {
	// Inject the fixture graph, and test data filtering on count operations
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(waitForIndexing)
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

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestSearchEdges() {
	// Inject the fixture graph, and test data filtering on count operations
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(waitForIndexing)
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

func (s *imageComponentCVEEdgeDatastoreSACTestSuite) TestSearchRawEdges() {
	// Inject the fixture graph, and test data filtering on count operations
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph(waitForIndexing)
	defer s.cleanImageToVulnerabilitiesGraph(waitForIndexing)
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
