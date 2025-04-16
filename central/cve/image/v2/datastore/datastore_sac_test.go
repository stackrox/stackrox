//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/suite"
)

var (
	log = logging.LoggerForModule()
)

func TestCVEV2DataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveV2DataStoreSACTestSuite))
}

type cveV2DataStoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore graphDBTestUtils.TestGraphDataStore
	imageCVEStore      DataStore

	imageTestContexts map[string]context.Context
}

func (s *cveV2DataStoreSACTestSuite) SetupSuite() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Skip("FlattenCVEData is disabled")
	}

	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.imageCVEStore = GetTestPostgresDataStore(s.T(), pool)
	s.Require().NoError(err)
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
}

// Vulnerability identifiers have been modified in the migration to Postgres to hold
// operating system information as well. This information is propagated from the image
// scan data.
// This helper is here to ease testing against the various datastore flavours.
func getImageCVEID(vuln *storage.EmbeddedVulnerability, component *storage.EmbeddedImageScanComponent, imageID string) string {
	componentID, _ := scancomponent.ComponentIDV2(component, imageID)
	cveID, _ := cve.IDV2(vuln, componentID)
	return cveID
}

func (s *cveV2DataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanImageToVulnerabilitiesGraph())
}

type cveTestCase struct {
	contextKey       string
	expectedCVEFound map[string]bool
}

var (
	imageCVETestCases = []cveTestCase{
		{
			contextKey: sacTestUtils.UnrestrictedReadCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": true,
				"CVE-4567-0002": true,
				"CVE-1234-0003": true,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": true,
				"CVE-2345-0007": true,
			},
		},
		{
			contextKey: sacTestUtils.UnrestrictedReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": true,
				"CVE-4567-0002": true,
				"CVE-1234-0003": true,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": true,
				"CVE-2345-0007": true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": true,
				"CVE-4567-0002": true,
				"CVE-1234-0003": true,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": true,
				"CVE-4567-0002": true,
				"CVE-1234-0003": true,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": false,
				"CVE-1234-0003": false,
				"CVE-3456-0004": false,
				"CVE-3456-0005": false,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": true,
				"CVE-4567-0002": true,
				"CVE-1234-0003": true,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": false,
				"CVE-1234-0003": false,
				"CVE-3456-0004": false,
				"CVE-3456-0005": false,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": true,
				"CVE-1234-0003": false,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": true,
				"CVE-2345-0007": true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": false,
				"CVE-1234-0003": false,
				"CVE-3456-0004": false,
				"CVE-3456-0005": false,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": true,
				"CVE-1234-0003": false,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": true,
				"CVE-2345-0007": true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": false,
				"CVE-1234-0003": false,
				"CVE-3456-0004": false,
				"CVE-3456-0005": false,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": true,
				"CVE-1234-0003": false,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": true,
				"CVE-2345-0007": true,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": false,
				"CVE-1234-0003": false,
				"CVE-3456-0004": false,
				"CVE-3456-0005": false,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": false,
				"CVE-1234-0003": false,
				"CVE-3456-0004": false,
				"CVE-3456-0005": false,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": false,
				"CVE-1234-0003": false,
				"CVE-3456-0004": false,
				"CVE-3456-0005": false,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": false,
				"CVE-4567-0002": false,
				"CVE-1234-0003": false,
				"CVE-3456-0004": false,
				"CVE-3456-0005": false,
				"CVE-2345-0006": false,
				"CVE-2345-0007": false,
			},
		},
		{
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
			// Therefore it should see all vulnerabilities.
			// (images are in cluster1 namespaceA and cluster2 namespaceB).
			expectedCVEFound: map[string]bool{
				"CVE-1234-0001": true,
				"CVE-4567-0002": true,
				"CVE-1234-0003": true,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": true,
				"CVE-2345-0007": true,
			},
		},
	}

	imageCVEByIDMap = map[string]*storage.EmbeddedVulnerability{
		getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   fixtures.GetEmbeddedImageCVE1234x0001(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()):   fixtures.GetEmbeddedImageCVE4567x0002(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003(), fixtures.GetEmbeddedImageComponent1x2(), fixtures.GetImageSherlockHolmes1().GetId()):   fixtures.GetEmbeddedImageCVE1234x0003(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): fixtures.GetEmbeddedImageCVE3456x0004(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005(), fixtures.GetEmbeddedImageComponent1s2x3(), fixtures.GetImageSherlockHolmes1().GetId()): fixtures.GetEmbeddedImageCVE3456x0005(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     fixtures.GetEmbeddedImageCVE2345x0006(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007(), fixtures.GetEmbeddedImageComponent2x5(), fixtures.GetImageDoctorJekyll2().GetId()):     fixtures.GetEmbeddedImageCVE2345x0007(),
	}
)

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVEExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for CVE-1234-0001
	targetCVE := fixtures.GetEmbeddedImageCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId())
	s.runImageTest("TestSACImageCVEExistsSingleScopeOnly", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageCVEStore.Exists(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], exists)
	})
}

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	targetCVE := fixtures.GetEmbeddedImageCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001(), fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId())
	cvss := targetCVE.GetCvss()
	s.runImageTest("TestSACImageCVEGetSingleScopeOnly", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		imageCVE, found, err := s.imageCVEStore.Get(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], found)
		if c.expectedCVEFound[cveName] {
			s.Require().NotNil(imageCVE)
			s.Equal(cveName, imageCVE.GetCveBaseInfo().GetCve())
			s.Equal(cvss, imageCVE.Cvss)
		} else {
			s.Nil(imageCVE)
		}
	})
}

func (s *cveV2DataStoreSACTestSuite) TestSACImageCVEGetBatch() {
	targetCVE1 := fixtures.GetEmbeddedImageCVE1234x0001()
	targetCVE2 := fixtures.GetEmbeddedImageCVE4567x0002()
	targetCVE3 := fixtures.GetEmbeddedImageCVE1234x0003()
	targetCVE4 := fixtures.GetEmbeddedImageCVE3456x0004()
	targetCVE6 := fixtures.GetEmbeddedImageCVE2345x0006()
	batchCVEs := []*storage.EmbeddedVulnerability{
		targetCVE1,
		targetCVE2,
		targetCVE3,
		targetCVE4,
		targetCVE6,
	}
	cveIDs := make([]string, 0, len(batchCVEs))
	for _, cve := range batchCVEs {
		cveIDs = append(cveIDs, getImageCVEID(cve, fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()))
	}
	s.runImageTest("TestSACImageCVEGetBatch", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		imageCVEs, err := s.imageCVEStore.GetBatch(testCtx, cveIDs)
		s.NoError(err)
		expectedCVEIDs := make([]string, 0, len(cveIDs))
		for _, cve := range batchCVEs {
			if c.expectedCVEFound[cve.GetCve()] {
				expectedCVEIDs = append(expectedCVEIDs, getImageCVEID(cve, fixtures.GetEmbeddedImageComponent1x1(), fixtures.GetImageSherlockHolmes1().GetId()))
			}
		}
		fetchedCVEIDs := make([]string, 0, len(imageCVEs))
		for _, imageCVE := range imageCVEs {
			fetchedCVEIDs = append(fetchedCVEIDs, imageCVE.GetId())
		}
		s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
	})
}

//func (s *cveV2DataStoreSACTestSuite) TestSACImageCVESearch() {
//	s.runImageTest("TestSACImageCVESearch", func(c cveTestCase) {
//
//		testCtx := s.imageTestContexts[c.contextKey]
//		results, err := s.imageCVEStore.Search(testCtx, nil)
//		s.NoError(err)
//		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
//		for name, visible := range c.expectedCVEFound {
//			if visible {
//				expectedCVENames = append(expectedCVENames, getImageCVEID(name))
//			}
//		}
//		fetchedCVEIDs := make(map[string]search.Result, 0)
//		for _, result := range results {
//			fetchedCVEIDs[result.ID] = result
//		}
//		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
//		for id := range fetchedCVEIDs {
//			fetchedCVENames = append(fetchedCVENames, id)
//		}
//		s.ElementsMatch(fetchedCVENames, expectedCVENames)
//	})
//}

//func (s *cveV2DataStoreSACTestSuite) TestSACImageCVESearchCVEs() {
//	s.runImageTest("TestSACImageCVESearchCVEs", func(c cveTestCase) {
//
//		testCtx := s.imageTestContexts[c.contextKey]
//		results, err := s.imageCVEStore.SearchImageCVEs(testCtx, nil)
//		s.NoError(err)
//		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
//		for name, visible := range c.expectedCVEFound {
//			if visible {
//				expectedCVENames = append(expectedCVENames, getImageCVEID(name))
//			}
//		}
//		fetchedCVEIDs := make(map[string]*v1.SearchResult, 0)
//		for _, result := range results {
//			fetchedCVEIDs[result.GetId()] = result
//		}
//		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
//		for id := range fetchedCVEIDs {
//			fetchedCVENames = append(fetchedCVENames, id)
//		}
//		s.ElementsMatch(fetchedCVENames, expectedCVENames)
//	})
//}

//func (s *cveV2DataStoreSACTestSuite) TestSACImageCVESearchRawCVEs() {
//	s.runImageTest("TestSACImageCVESearchRawCVEs", func(c cveTestCase) {
//
//		testCtx := s.imageTestContexts[c.contextKey]
//		results, err := s.imageCVEStore.SearchRawImageCVEs(testCtx, nil)
//		s.NoError(err)
//		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
//		for name, visible := range c.expectedCVEFound {
//			if visible {
//				expectedCVENames = append(expectedCVENames, getImageCVEID(name))
//			}
//		}
//		fetchedCVEIDs := make(map[string]*storage.ImageCVEV2, 0)
//		for _, result := range results {
//			fetchedCVEIDs[result.GetId()] = result
//			s.Equal(imageCVEByIDMap[result.GetId()].GetCvss(), result.GetCvss())
//		}
//		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
//		for id := range fetchedCVEIDs {
//			fetchedCVENames = append(fetchedCVENames, id)
//		}
//		s.ElementsMatch(fetchedCVENames, expectedCVENames)
//	})
//}

func (s *cveV2DataStoreSACTestSuite) runImageTest(testName string, testFunc func(c cveTestCase)) {
	err := s.testGraphDatastore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)

	imageGraphBefore := graphDBTestUtils.GetImageGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)

	failed := false
	for _, c := range imageCVETestCases {
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
