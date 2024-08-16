//go:build sql_integration

package datastoretest

import (
	"context"
	"testing"

	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

var (
	log = logging.LoggerForModule()
)

func TestCVEDataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveDataStoreSACTestSuite))
}

type cveDataStoreSACTestSuite struct {
	suite.Suite

	testGraphDatastore graphDBTestUtils.TestGraphDataStore
	imageCVEStore      imageCVEDataStore.DataStore
	nodeCVEStore       nodeCVEDataStore.DataStore

	nodeTestContexts  map[string]context.Context
	imageTestContexts map[string]context.Context
}

func (s *cveDataStoreSACTestSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	pool := s.testGraphDatastore.GetPostgresPool()
	s.imageCVEStore = imageCVEDataStore.GetTestPostgresDataStore(s.T(), pool)
	s.nodeCVEStore, err = nodeCVEDataStore.GetTestPostgresDataStore(s.T(), pool)
	s.Require().NoError(err)
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
	s.nodeTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
}

func (s *cveDataStoreSACTestSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

// Vulnerability identifiers have been modified in the migration to Postgres to hold
// operating system information as well. This information is propagated from the image
// scan data.
// This helper is here to ease testing against the various datastore flavours.
func getImageCVEID(cve string) string {
	return cve + "#crime-stories"
}

// Vulnerability identifiers have been modified in the migration to Postgres to hold
// operating system information as well. This information is propagated from the node
// scan data.
// This helper is here to ease testing against the various datastore flavours.
func getNodeCVEID(cve string) string {
	return cve + "#Linux"
}

func (s *cveDataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanImageToVulnerabilitiesGraph())
}

func (s *cveDataStoreSACTestSuite) cleanNodeToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanNodeToVulnerabilitiesGraph())
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
		getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0001().GetCve()): fixtures.GetEmbeddedImageCVE1234x0001(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE4567x0002().GetCve()): fixtures.GetEmbeddedImageCVE4567x0002(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE1234x0003().GetCve()): fixtures.GetEmbeddedImageCVE1234x0003(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0004().GetCve()): fixtures.GetEmbeddedImageCVE3456x0004(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE3456x0005().GetCve()): fixtures.GetEmbeddedImageCVE3456x0005(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0006().GetCve()): fixtures.GetEmbeddedImageCVE2345x0006(),
		getImageCVEID(fixtures.GetEmbeddedImageCVE2345x0007().GetCve()): fixtures.GetEmbeddedImageCVE2345x0007(),
	}

	nodeCVETestCases = []cveTestCase{
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
			contextKey: sacTestUtils.MixedClusterAndNamespaceReadCtx,
			// The mixed scope context can see cluster1 and namespaceA as well as all cluster2.
			// Therefore it should see only cluster2 vulnerabilities (and shared ones).
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

	nodeCVEByIDMap = map[string]*storage.EmbeddedVulnerability{
		getNodeCVEID(fixtures.GetEmbeddedNodeCVE1234x0001().GetCve()): fixtures.GetEmbeddedNodeCVE1234x0001(),
		getNodeCVEID(fixtures.GetEmbeddedNodeCVE4567x0002().GetCve()): fixtures.GetEmbeddedNodeCVE4567x0002(),
		getNodeCVEID(fixtures.GetEmbeddedNodeCVE1234x0003().GetCve()): fixtures.GetEmbeddedNodeCVE1234x0003(),
		getNodeCVEID(fixtures.GetEmbeddedNodeCVE3456x0004().GetCve()): fixtures.GetEmbeddedNodeCVE3456x0004(),
		getNodeCVEID(fixtures.GetEmbeddedNodeCVE3456x0005().GetCve()): fixtures.GetEmbeddedNodeCVE3456x0005(),
		getNodeCVEID(fixtures.GetEmbeddedNodeCVE2345x0006().GetCve()): fixtures.GetEmbeddedNodeCVE2345x0006(),
		getNodeCVEID(fixtures.GetEmbeddedNodeCVE2345x0007().GetCve()): fixtures.GetEmbeddedNodeCVE2345x0007(),
	}
)

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for CVE-1234-0001
	targetCVE := fixtures.GetEmbeddedImageCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(cveName)
	s.runImageTest("TestSACImageCVEExistsSingleScopeOnly", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageCVEStore.Exists(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsSharedAcrossComponents() {
	// Inject the fixture graph, and test exists for CVE-4567-0002
	targetCVE := fixtures.GetEmbeddedImageCVE4567x0002()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(cveName)
	s.runImageTest("TestSACImageCVEExistsSharedAcrossComponents", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageCVEStore.Exists(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsFromSharedComponent() {
	// Inject the fixture graph, and test exists for CVE-3456-0004
	targetCVE := fixtures.GetEmbeddedImageCVE3456x0004()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(cveName)
	s.runImageTest("TestSACImageCVEExistsFromSharedComponent", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		exists, err := s.imageCVEStore.Exists(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	targetCVE := fixtures.GetEmbeddedImageCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(cveName)
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

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetSharedAcrossComponents() {
	// Inject the fixture graph, and test retrieval for CVE-4567-0002
	targetCVE := fixtures.GetEmbeddedImageCVE4567x0002()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(cveName)
	cvss := targetCVE.GetCvss()
	s.runImageTest("TestSACImageCVEGetSharedAcrossComponents", func(c cveTestCase) {
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

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetFromSharedComponent() {
	// Inject the fixture graph, and test retrieval for CVE-3456-0004
	targetCVE := fixtures.GetEmbeddedImageCVE3456x0004()
	cveName := targetCVE.GetCve()
	cveID := getImageCVEID(cveName)
	cvss := targetCVE.GetCvss()
	s.runImageTest("TestSACImageCVEGetFromSharedComponent", func(c cveTestCase) {
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

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetBatch() {
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
		cveIDs = append(cveIDs, getImageCVEID(cve.GetCve()))
	}
	s.runImageTest("TestSACImageCVEGetBatch", func(c cveTestCase) {
		testCtx := s.imageTestContexts[c.contextKey]
		imageCVEs, err := s.imageCVEStore.GetBatch(testCtx, cveIDs)
		s.NoError(err)
		expectedCVEIDs := make([]string, 0, len(cveIDs))
		for _, cve := range batchCVEs {
			if c.expectedCVEFound[cve.GetCve()] {
				expectedCVEIDs = append(expectedCVEIDs, getImageCVEID(cve.GetCve()))
			}
		}
		fetchedCVEIDs := make([]string, 0, len(imageCVEs))
		for _, imageCVE := range imageCVEs {
			fetchedCVEIDs = append(fetchedCVEIDs, imageCVE.GetId())
		}
		s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearch() {
	s.runImageTest("TestSACImageCVESearch", func(c cveTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageCVEStore.Search(testCtx, nil)
		s.NoError(err)
		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVENames = append(expectedCVENames, getImageCVEID(name))
			}
		}
		fetchedCVEIDs := make(map[string]search.Result, 0)
		for _, result := range results {
			fetchedCVEIDs[result.ID] = result
		}
		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
		for id := range fetchedCVEIDs {
			fetchedCVENames = append(fetchedCVENames, id)
		}
		s.ElementsMatch(fetchedCVENames, expectedCVENames)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearchCVEs() {
	s.runImageTest("TestSACImageCVESearchCVEs", func(c cveTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageCVEStore.SearchImageCVEs(testCtx, nil)
		s.NoError(err)
		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVENames = append(expectedCVENames, getImageCVEID(name))
			}
		}
		fetchedCVEIDs := make(map[string]*v1.SearchResult, 0)
		for _, result := range results {
			fetchedCVEIDs[result.GetId()] = result
		}
		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
		for id := range fetchedCVEIDs {
			fetchedCVENames = append(fetchedCVENames, id)
		}
		s.ElementsMatch(fetchedCVENames, expectedCVENames)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearchRawCVEs() {
	s.runImageTest("TestSACImageCVESearchRawCVEs", func(c cveTestCase) {

		testCtx := s.imageTestContexts[c.contextKey]
		results, err := s.imageCVEStore.SearchRawImageCVEs(testCtx, nil)
		s.NoError(err)
		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVENames = append(expectedCVENames, getImageCVEID(name))
			}
		}
		fetchedCVEIDs := make(map[string]*storage.ImageCVE, 0)
		for _, result := range results {
			fetchedCVEIDs[result.GetId()] = result
			s.Equal(imageCVEByIDMap[result.GetId()].GetCvss(), result.GetCvss())
		}
		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
		for id := range fetchedCVEIDs {
			fetchedCVENames = append(fetchedCVENames, id)
		}
		s.ElementsMatch(fetchedCVENames, expectedCVENames)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESuppress() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEUnsuppress() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACEnrichImageWithSuppressedCVEs() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) runImageTest(testName string, testFunc func(c cveTestCase)) {
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

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for CVE-1234-0001
	targetCVE := fixtures.GetEmbeddedNodeCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := getNodeCVEID(cveName)
	s.runNodeTest("TestSACNodeCVEExistsSingleScopeOnly", func(c cveTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		exists, err := s.nodeCVEStore.Exists(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsSharedAcrossComponents() {
	// Inject the fixture graph, and test exists for CVE-4567-0002
	targetCVE := fixtures.GetEmbeddedNodeCVE4567x0002()
	cveName := targetCVE.GetCve()
	cveID := getNodeCVEID(cveName)
	s.runNodeTest("TestSACNodeCVEExistsSharedAcrossComponents", func(c cveTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		exists, err := s.nodeCVEStore.Exists(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsFromSharedComponent() {
	// Inject the fixture graph, and test exists for CVE-3456-0004
	targetCVE := fixtures.GetEmbeddedNodeCVE3456x0004()
	cveName := targetCVE.GetCve()
	cveID := getNodeCVEID(cveName)
	s.runNodeTest("TestSACNodeCVEExistsFromSharedComponent", func(c cveTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		exists, err := s.nodeCVEStore.Exists(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], exists)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	targetCVE := fixtures.GetEmbeddedNodeCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := getNodeCVEID(cveName)
	cvss := targetCVE.GetCvss()
	s.runNodeTest("TestSACNodeCVEGetSingleScopeOnly", func(c cveTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		nodeCVE, found, err := s.nodeCVEStore.Get(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], found)
		if c.expectedCVEFound[cveName] {
			s.Require().NotNil(nodeCVE)
			s.Equal(cveName, nodeCVE.GetCveBaseInfo().GetCve())
			s.Equal(cvss, nodeCVE.Cvss)
		} else {
			s.Nil(nodeCVE)
		}
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetSharedAcrossComponents() {
	// Inject the fixture graph, and test retrieval for CVE-4567-0002
	targetCVE := fixtures.GetEmbeddedNodeCVE4567x0002()
	cveName := targetCVE.GetCve()
	cveID := getNodeCVEID(cveName)
	cvss := targetCVE.GetCvss()
	s.runNodeTest("TestSACNodeCVEGetSharedAcrossComponents", func(c cveTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		nodeCVE, found, err := s.nodeCVEStore.Get(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], found)
		if c.expectedCVEFound[cveName] {
			s.Require().NotNil(nodeCVE)
			s.Equal(cveName, nodeCVE.GetCveBaseInfo().GetCve())
			s.Equal(cvss, nodeCVE.Cvss)
		} else {
			s.Nil(nodeCVE)
		}
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetFromSharedComponent() {
	// Inject the fixture graph, and test retrieval for CVE-3456-0004
	targetCVE := fixtures.GetEmbeddedNodeCVE3456x0004()
	cveName := targetCVE.GetCve()
	cveID := getNodeCVEID(cveName)
	cvss := targetCVE.GetCvss()
	s.runNodeTest("TestSACNodeCVEGetFromSharedComponent", func(c cveTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		nodeCVE, found, err := s.nodeCVEStore.Get(testCtx, cveID)
		s.NoError(err)
		s.Equal(c.expectedCVEFound[cveName], found)
		if c.expectedCVEFound[cveName] {
			s.Require().NotNil(nodeCVE)
			s.Equal(cveName, nodeCVE.GetCveBaseInfo().GetCve())
			s.Equal(cvss, nodeCVE.Cvss)
		} else {
			s.Nil(nodeCVE)
		}
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetBatch() {
	targetCVE1 := fixtures.GetEmbeddedNodeCVE1234x0001()
	targetCVE2 := fixtures.GetEmbeddedNodeCVE4567x0002()
	targetCVE3 := fixtures.GetEmbeddedNodeCVE1234x0003()
	targetCVE4 := fixtures.GetEmbeddedNodeCVE3456x0004()
	targetCVE6 := fixtures.GetEmbeddedNodeCVE2345x0006()
	batchCVEs := []*storage.EmbeddedVulnerability{
		targetCVE1,
		targetCVE2,
		targetCVE3,
		targetCVE4,
		targetCVE6,
	}
	cveIDs := make([]string, 0, len(batchCVEs))
	for _, cve := range batchCVEs {
		cveIDs = append(cveIDs, getNodeCVEID(cve.GetCve()))
	}
	s.runNodeTest("TestSACNodeCVEGetBatch", func(c cveTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		nodeCVEs, err := s.nodeCVEStore.GetBatch(testCtx, cveIDs)
		s.NoError(err)
		expectedCVEIDs := make([]string, 0, len(batchCVEs))
		for _, cve := range batchCVEs {
			if c.expectedCVEFound[cve.GetCve()] {
				expectedCVEIDs = append(expectedCVEIDs, getNodeCVEID(cve.GetCve()))
			}
		}
		fetchedCVEIDs := make([]string, 0, len(nodeCVEs))
		for _, nodeCVE := range nodeCVEs {
			fetchedCVEIDs = append(fetchedCVEIDs, nodeCVE.GetId())
		}
		s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVECount() {
	s.runNodeTest("TestSACNodeCVECount", func(c cveTestCase) {
		testCtx := s.nodeTestContexts[c.contextKey]
		count, err := s.nodeCVEStore.Count(testCtx, nil)
		s.NoError(err)
		expectedCount := 0
		for _, visible := range c.expectedCVEFound {
			if visible {
				expectedCount++
			}
		}
		s.Equal(expectedCount, count)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearch() {
	s.runNodeTest("TestSACNodeCVESearch", func(c cveTestCase) {

		testCtx := s.nodeTestContexts[c.contextKey]
		results, err := s.nodeCVEStore.Search(testCtx, nil)
		s.NoError(err)
		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVENames = append(expectedCVENames, getNodeCVEID(name))
			}
		}
		fetchedCVEIDs := make(map[string]search.Result, 0)
		for _, result := range results {
			fetchedCVEIDs[result.ID] = result
		}
		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
		for id := range fetchedCVEIDs {
			fetchedCVENames = append(fetchedCVENames, id)
		}
		s.ElementsMatch(fetchedCVENames, expectedCVENames)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearchCVEs() {
	s.runNodeTest("TestSACNodeCVESearchCVEs", func(c cveTestCase) {

		testCtx := s.nodeTestContexts[c.contextKey]
		results, err := s.nodeCVEStore.SearchNodeCVEs(testCtx, nil)
		s.NoError(err)
		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVENames = append(expectedCVENames, getNodeCVEID(name))
			}
		}
		fetchedCVEIDs := make(map[string]*v1.SearchResult, 0)
		for _, result := range results {
			fetchedCVEIDs[result.GetId()] = result
		}
		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
		for id := range fetchedCVEIDs {
			fetchedCVENames = append(fetchedCVENames, id)
		}
		s.ElementsMatch(fetchedCVENames, expectedCVENames)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearchRawCVEs() {
	s.runNodeTest("TestSACNodeCVESearchRawCVEs", func(c cveTestCase) {

		testCtx := s.nodeTestContexts[c.contextKey]
		results, err := s.nodeCVEStore.SearchRawCVEs(testCtx, nil)
		s.NoError(err)
		expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
		for name, visible := range c.expectedCVEFound {
			if visible {
				expectedCVENames = append(expectedCVENames, getNodeCVEID(name))
			}
		}
		fetchedCVEIDs := make(map[string]*storage.NodeCVE, 0)
		for _, result := range results {
			fetchedCVEIDs[result.GetId()] = result
			s.Equal(nodeCVEByIDMap[result.GetId()].GetCvss(), result.GetCvss())
		}
		fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
		for id := range fetchedCVEIDs {
			fetchedCVENames = append(fetchedCVENames, id)
		}
		s.ElementsMatch(fetchedCVENames, expectedCVENames)
	})
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESuppress() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEUnsuppress() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACEnrichNodeWithSuppressedCVEs() {
	s.T().Skip("Not implemented yet.")

}

func (s *cveDataStoreSACTestSuite) runNodeTest(testName string, testFunc func(c cveTestCase)) {
	err := s.testGraphDatastore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	s.Require().NoError(err)

	nodeGraphBefore := graphDBTestUtils.GetNodeGraph(
		sac.WithAllAccess(context.Background()),
		s.T(),
		s.testGraphDatastore.GetPostgresPool(),
	)

	failed := false
	for _, c := range nodeCVETestCases {
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
