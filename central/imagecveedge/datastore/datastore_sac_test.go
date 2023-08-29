//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/cve/converter/utils"
	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/suite"
)

const (
	imageOS = "crime-stories"
)

var (
	log = logging.LoggerForModule()
)

func TestImageCVEEdgeDataStoreSAC(t *testing.T) {
	suite.Run(t, new(imageCVEEdgeDatastoreSACTestSuite))
}

type imageCVEEdgeDatastoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore graphDBTestUtils.TestGraphDataStore
	datastore          DataStore

	testContexts map[string]context.Context
}

func (s *imageCVEEdgeDatastoreSACTestSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.datastore = GetTestPostgresDataStore(s.T(), pool)
	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

func (s *imageCVEEdgeDatastoreSACTestSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

func (s *imageCVEEdgeDatastoreSACTestSuite) SetupTest() {
	s.Require().NoError(s.testGraphDatastore.PushImageToVulnerabilitiesGraph())
}

func (s *imageCVEEdgeDatastoreSACTestSuite) TearDownTest() {
	s.cleanImageToVulnerabilitiesGraph()
}

func (s *imageCVEEdgeDatastoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	now := time.Now()
	fmt.Println(now.UnixMicro(), " - Cleaning up")
	s.Require().NoError(s.testGraphDatastore.CleanImageToVulnerabilitiesGraph())
}

func getCveID(vulnerability *storage.EmbeddedVulnerability, os string) string {
	return utils.EmbeddedCVEToProtoCVE(os, vulnerability).GetId()
}

func getEdgeID(image *storage.Image, vulnerability *storage.EmbeddedVulnerability, os string) string {
	imageID := image.GetId()
	convertedCVEID := getCveID(vulnerability, os)
	return pgSearch.IDFromPks([]string{imageID, convertedCVEID})
}

type edgeTestCase struct {
	contextKey        string
	expectedEdgeFound map[string]bool
}

var (
	img1cve1edge = getEdgeID(fixtures.GetImageSherlockHolmes1(), fixtures.GetEmbeddedImageCVE1234x0001(), imageOS)
	img1cve2edge = getEdgeID(fixtures.GetImageSherlockHolmes1(), fixtures.GetEmbeddedImageCVE4567x0002(), imageOS)
	img1cve3edge = getEdgeID(fixtures.GetImageSherlockHolmes1(), fixtures.GetEmbeddedImageCVE1234x0003(), imageOS)
	img1cve4edge = getEdgeID(fixtures.GetImageSherlockHolmes1(), fixtures.GetEmbeddedImageCVE3456x0004(), imageOS)
	img1cve5edge = getEdgeID(fixtures.GetImageSherlockHolmes1(), fixtures.GetEmbeddedImageCVE3456x0005(), imageOS)
	img2cve2edge = getEdgeID(fixtures.GetImageDoctorJekyll2(), fixtures.GetEmbeddedImageCVE4567x0002(), imageOS)
	img2cve4edge = getEdgeID(fixtures.GetImageDoctorJekyll2(), fixtures.GetEmbeddedImageCVE3456x0004(), imageOS)
	img2cve5edge = getEdgeID(fixtures.GetImageDoctorJekyll2(), fixtures.GetEmbeddedImageCVE3456x0005(), imageOS)
	img2cve6edge = getEdgeID(fixtures.GetImageDoctorJekyll2(), fixtures.GetEmbeddedImageCVE2345x0006(), imageOS)
	img2cve7edge = getEdgeID(fixtures.GetImageDoctorJekyll2(), fixtures.GetEmbeddedImageCVE2345x0007(), imageOS)

	fullAccessMap = map[string]bool{
		img1cve1edge: true,
		img1cve2edge: true,
		img1cve3edge: true,
		img1cve4edge: true,
		img1cve5edge: true,
		img2cve2edge: true,
		img2cve4edge: true,
		img2cve5edge: true,
		img2cve6edge: true,
		img2cve7edge: true,
	}

	cluster1WithNamespaceAMap = map[string]bool{
		img1cve1edge: true,
		img1cve2edge: true,
		img1cve3edge: true,
		img1cve4edge: true,
		img1cve5edge: true,
		img2cve2edge: false,
		img2cve4edge: false,
		img2cve5edge: false,
		img2cve6edge: false,
		img2cve7edge: false,
	}

	cluster2WithNamespaceBMap = map[string]bool{
		img1cve1edge: false,
		img1cve2edge: false,
		img1cve3edge: false,
		img1cve4edge: false,
		img1cve5edge: false,
		img2cve2edge: true,
		img2cve4edge: true,
		img2cve5edge: true,
		img2cve6edge: true,
		img2cve7edge: true,
	}

	noAccessMap = map[string]bool{
		img1cve1edge: false,
		img1cve2edge: false,
		img1cve3edge: false,
		img1cve4edge: false,
		img1cve5edge: false,
		img2cve2edge: false,
		img2cve4edge: false,
		img2cve5edge: false,
		img2cve6edge: false,
		img2cve7edge: false,
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

func (s *imageCVEEdgeDatastoreSACTestSuite) TestGet() {
	// Inject the fixture graph, and test exists for Image1 to CVE-1234-0001 edge
	targetEdgeID := img1cve1edge
	expectedSrcID := fixtures.GetImageSherlockHolmes1().GetId()
	expectedTargetID := getCveID(fixtures.GetEmbeddedImageCVE1234x0001(), imageOS)
	s.run("TestGet", func(c edgeTestCase) {
		testCtx := s.testContexts[c.contextKey]
		edge, exists, err := s.datastore.Get(testCtx, targetEdgeID)
		s.NoError(err)
		if c.expectedEdgeFound[targetEdgeID] {
			s.True(exists)
			s.NotNil(edge)
			s.Equal(expectedSrcID, edge.GetImageId())
			s.Equal(expectedTargetID, edge.GetImageCveId())
		} else {
			s.False(exists)
			s.Nil(edge)
		}
	})
}

func (s *imageCVEEdgeDatastoreSACTestSuite) TestSearch() {
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
		s.Equal(expectedCount, len(results))
		for _, r := range results {
			s.True(c.expectedEdgeFound[r.ID])
		}
	})
}

func (s *imageCVEEdgeDatastoreSACTestSuite) TestSearchEdges() {
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
		s.Equal(expectedCount, len(results))
		for _, r := range results {
			s.True(c.expectedEdgeFound[r.GetId()])
		}
	})
}

func (s *imageCVEEdgeDatastoreSACTestSuite) TestSearchRawEdges() {
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
		s.Equal(expectedCount, len(results))
		for _, r := range results {
			s.True(c.expectedEdgeFound[r.GetId()])
		}
	})
}
func (s *imageCVEEdgeDatastoreSACTestSuite) run(testName string, testFunc func(c edgeTestCase)) {
	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)

	failed := false
	for _, c := range testCases {
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
		imageGraphBefore.Log()
	}
}
