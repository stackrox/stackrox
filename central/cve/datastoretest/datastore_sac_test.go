package datastoretest

import (
	"context"
	"testing"
	"time"

	genericCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	dackboxTestUtils "github.com/stackrox/rox/central/dackbox/testutils"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stretchr/testify/suite"
)

func TestCVEDataStoreSAC(t *testing.T) {
	suite.Run(t, new(cveDataStoreSACTestSuite))
}

type cveDataStoreSACTestSuite struct {
	suite.Suite

	dackboxTestStore dackboxTestUtils.DackboxTestDataStore
	imageCVEStore    imageCVEDataStore.DataStore
	nodeCVEStore     nodeCVEDataStore.DataStore

	nodeTestContexts  map[string]context.Context
	imageTestContexts map[string]context.Context
}

func (s *cveDataStoreSACTestSuite) SetupSuite() {
	var err error
	s.dackboxTestStore, err = dackboxTestUtils.NewDackboxTestDataStore(s.T())
	s.Require().NoError(err)
	if features.PostgresDatastore.Enabled() {
		pool := s.dackboxTestStore.GetPostgresPool()
		s.imageCVEStore, err = imageCVEDataStore.GetTestPostgresDataStore(s.T(), pool)
		s.Require().NoError(err)
		s.nodeCVEStore, err = nodeCVEDataStore.GetTestPostgresDataStore(s.T(), pool)
		s.Require().NoError(err)
	} else {
		dacky := s.dackboxTestStore.GetDackbox()
		keyFence := s.dackboxTestStore.GetKeyFence()
		rocksEngine := s.dackboxTestStore.GetRocksEngine()
		bleveIndex := s.dackboxTestStore.GetBleveIndex()
		indexQ := s.dackboxTestStore.GetIndexQ()
		genericCVEStore, err := genericCVEDataStore.GetTestRocksBleveDataStore(s.T(), rocksEngine, bleveIndex, dacky,
			keyFence, indexQ)
		s.Require().NoError(err)
		s.imageCVEStore = &imageCVEDataStoreFromGenericStore{
			genericStore: genericCVEStore,
		}
		s.nodeCVEStore = &nodeCVEDataStoreFromGenericStore{
			genericStore: genericCVEStore,
		}
	}
	s.imageTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
	s.nodeTestContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Node)
}

func (s *cveDataStoreSACTestSuite) TearDownSuite() {
	s.Require().NoError(s.dackboxTestStore.Cleanup(s.T()))
}

// Vulnerability identifiers have been modified in the migration to Postgres to hold
// operating system information as well. This information is propagated from the image
// scan data.
// This helper is here to ease testing against the various datastore flavours.
func (s *cveDataStoreSACTestSuite) getImageCVEID(cve string) string {
	if features.PostgresDatastore.Enabled() {
		return cve + "#crime-stories"
	}
	return cve
}

// Vulnerability identifiers have been modified in the migration to Postgres to hold
// operating system information as well. This information is propagated from the node
// scan data.
// This helper is here to ease testing against the various datastore flavours.
func (s *cveDataStoreSACTestSuite) getNodeCVEID(cve string) string {
	if features.PostgresDatastore.Enabled() {
		return cve + "#Linux"
	}
	return cve
}

func (s *cveDataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph(wait bool) {
	s.Require().NoError(s.dackboxTestStore.CleanImageToVulnerabilitiesGraph())
	if wait {
		time.Sleep(1 * time.Second)
	}
}

func (s *cveDataStoreSACTestSuite) cleanNodeToVulnerabilitiesGraph(wait bool) {
	s.Require().NoError(s.dackboxTestStore.CleanNodeToVulnerabilitiesGraph())
	if wait {
		time.Sleep(1 * time.Second)
	}
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
			// Partial cluster scope is too narrow for allowfixedscope at cluster level.
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
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			// Partial cluster scope is too narrow for allowfixedscope at cluster level.
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
				"CVE-1234-0001": false,
				"CVE-4567-0002": true,
				"CVE-1234-0003": false,
				"CVE-3456-0004": true,
				"CVE-3456-0005": true,
				"CVE-2345-0006": true,
				"CVE-2345-0007": true,
			},
		},
	}
)

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for CVE-1234-0001
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := s.getImageCVEID(cveName)
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			exists, err := s.imageCVEStore.Exists(testCtx, cveID)
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveName], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsSharedAcrossComponents() {
	// Inject the fixture graph, and test exists for CVE-4567-0002
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE4567x0002()
	cveName := targetCVE.GetCve()
	cveID := s.getImageCVEID(cveName)
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			exists, err := s.imageCVEStore.Exists(testCtx, cveID)
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveName], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsFromSharedComponent() {
	// Inject the fixture graph, and test exists for CVE-3456-0004
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE3456x0004()
	cveName := targetCVE.GetCve()
	cveID := s.getImageCVEID(cveName)
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			exists, err := s.imageCVEStore.Exists(testCtx, cveID)
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveName], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := s.getImageCVEID(cveName)
	cvss := targetCVE.GetCvss()
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
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
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetSharedAcrossComponents() {
	// Inject the fixture graph, and test retrieval for CVE-4567-0002
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE4567x0002()
	cveName := targetCVE.GetCve()
	cveID := s.getImageCVEID(cveName)
	cvss := targetCVE.GetCvss()
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
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
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetFromSharedComponent() {
	// Inject the fixture graph, and test retrieval for CVE-3456-0004
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedImageCVE3456x0004()
	cveName := targetCVE.GetCve()
	cveID := s.getImageCVEID(cveName)
	cvss := targetCVE.GetCvss()
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
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
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetBatch() {
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
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
		cveIDs = append(cveIDs, s.getImageCVEID(cve.GetCve()))
	}
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.imageTestContexts[c.contextKey]
			imageCVEs, err := s.imageCVEStore.GetBatch(testCtx, cveIDs)
			s.NoError(err)
			expectedCVEIDs := make([]string, 0, len(cveIDs))
			for _, cve := range batchCVEs {
				if c.expectedCVEFound[cve.GetCve()] {
					expectedCVEIDs = append(expectedCVEIDs, s.getImageCVEID(cve.GetCve()))
				}
			}
			fetchedCVEIDs := make([]string, 0, len(imageCVEs))
			for _, imageCVE := range imageCVEs {
				fetchedCVEIDs = append(fetchedCVEIDs, imageCVE.GetId())
			}
			s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVECount() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearch() {
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(!features.PostgresDatastore.Enabled())
	if !features.PostgresDatastore.Enabled() {
		// Wait for bleve indexing to complete
		time.Sleep(1 * time.Second)
	}
	s.Require().NoError(err)
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {

			testCtx := s.imageTestContexts[c.contextKey]
			results, err := s.imageCVEStore.Search(testCtx, nil)
			s.NoError(err)
			expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
			for name, visible := range c.expectedCVEFound {
				if visible {
					expectedCVENames = append(expectedCVENames, s.getImageCVEID(name))
				}
			}
			fetchedCVEIDs := make(map[string]bool, 0)
			for _, result := range results {
				fetchedCVEIDs[result.ID] = true
			}
			fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
			for id := range fetchedCVEIDs {
				fetchedCVENames = append(fetchedCVENames, id)
			}
			s.ElementsMatch(fetchedCVENames, expectedCVENames)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearchCVEs() {
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(!features.PostgresDatastore.Enabled())
	if !features.PostgresDatastore.Enabled() {
		// Wait for bleve indexing to complete
		time.Sleep(1 * time.Second)
	}
	s.Require().NoError(err)
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {

			testCtx := s.imageTestContexts[c.contextKey]
			results, err := s.imageCVEStore.SearchImageCVEs(testCtx, nil)
			s.NoError(err)
			expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
			for name, visible := range c.expectedCVEFound {
				if visible {
					expectedCVENames = append(expectedCVENames, s.getImageCVEID(name))
				}
			}
			fetchedCVEIDs := make(map[string]bool, 0)
			for _, result := range results {
				fetchedCVEIDs[result.GetId()] = true
			}
			fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
			for id := range fetchedCVEIDs {
				fetchedCVENames = append(fetchedCVENames, id)
			}
			s.ElementsMatch(fetchedCVENames, expectedCVENames)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearchRawCVEs() {
	err := s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph(!features.PostgresDatastore.Enabled())
	if !features.PostgresDatastore.Enabled() {
		// Wait for bleve indexing to complete
		time.Sleep(1 * time.Second)
	}
	s.Require().NoError(err)
	for _, c := range imageCVETestCases {
		s.Run(c.contextKey, func() {

			testCtx := s.imageTestContexts[c.contextKey]
			results, err := s.imageCVEStore.SearchRawImageCVEs(testCtx, nil)
			s.NoError(err)
			expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
			for name, visible := range c.expectedCVEFound {
				if visible {
					expectedCVENames = append(expectedCVENames, s.getImageCVEID(name))
				}
			}
			fetchedCVEIDs := make(map[string]bool, 0)
			for _, result := range results {
				fetchedCVEIDs[result.GetId()] = true
			}
			fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
			for id := range fetchedCVEIDs {
				fetchedCVENames = append(fetchedCVENames, id)
			}
			s.ElementsMatch(fetchedCVENames, expectedCVENames)
		})
	}
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

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for CVE-1234-0001
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedNodeCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := s.getNodeCVEID(cveName)
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.nodeTestContexts[c.contextKey]
			exists, err := s.nodeCVEStore.Exists(testCtx, cveID)
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveName], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsSharedAcrossComponents() {
	// Inject the fixture graph, and test exists for CVE-4567-0002
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedNodeCVE4567x0002()
	cveName := targetCVE.GetCve()
	cveID := s.getNodeCVEID(cveName)
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.nodeTestContexts[c.contextKey]
			exists, err := s.nodeCVEStore.Exists(testCtx, cveID)
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveName], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsFromSharedComponent() {
	// Inject the fixture graph, and test exists for CVE-3456-0004
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedNodeCVE3456x0004()
	cveName := targetCVE.GetCve()
	cveID := s.getNodeCVEID(cveName)
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.nodeTestContexts[c.contextKey]
			exists, err := s.nodeCVEStore.Exists(testCtx, cveID)
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveName], exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedNodeCVE1234x0001()
	cveName := targetCVE.GetCve()
	cveID := s.getNodeCVEID(cveName)
	cvss := targetCVE.GetCvss()
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeCVE, found, err := s.nodeCVEStore.Get(testCtx, cveID)
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveName], found)
			if c.expectedCVEFound[cveName] {
				s.NotNil(nodeCVE)
				s.Equal(cveName, nodeCVE.GetCveBaseInfo().GetCve())
				s.Equal(cvss, nodeCVE.Cvss)
			} else {
				s.Nil(nodeCVE)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetSharedAcrossComponents() {
	// Inject the fixture graph, and test retrieval for CVE-4567-0002
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedNodeCVE4567x0002()
	cveName := targetCVE.GetCve()
	cveID := s.getNodeCVEID(cveName)
	cvss := targetCVE.GetCvss()
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeCVE, found, err := s.nodeCVEStore.Get(testCtx, cveID)
			s.NoError(err)
			s.Equal(c.expectedCVEFound[cveName], found)
			if c.expectedCVEFound[cveName] {
				s.NotNil(nodeCVE)
				s.Equal(cveName, nodeCVE.GetCveBaseInfo().GetCve())
				s.Equal(cvss, nodeCVE.Cvss)
			} else {
				s.Nil(nodeCVE)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetFromSharedComponent() {
	// Inject the fixture graph, and test retrieval for CVE-3456-0004
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
	targetCVE := fixtures.GetEmbeddedNodeCVE3456x0004()
	cveName := targetCVE.GetCve()
	cveID := s.getNodeCVEID(cveName)
	cvss := targetCVE.GetCvss()
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {
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
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetBatch() {
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(false)
	s.Require().NoError(err)
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
		cveIDs = append(cveIDs, s.getNodeCVEID(cve.GetCve()))
	}
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeCVEs, err := s.nodeCVEStore.GetBatch(testCtx, cveIDs)
			s.NoError(err)
			expectedCVEIDs := make([]string, 0, len(batchCVEs))
			for _, cve := range batchCVEs {
				if c.expectedCVEFound[cve.GetCve()] {
					expectedCVEIDs = append(expectedCVEIDs, s.getNodeCVEID(cve.GetCve()))
				}
			}
			fetchedCVEIDs := make([]string, 0, len(nodeCVEs))
			for _, nodeCVE := range nodeCVEs {
				fetchedCVEIDs = append(fetchedCVEIDs, nodeCVE.GetId())
			}
			s.ElementsMatch(expectedCVEIDs, fetchedCVEIDs)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVECount() {
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(!features.PostgresDatastore.Enabled())
	if !features.PostgresDatastore.Enabled() {
		// Wait for bleve indexing to complete
		time.Sleep(1 * time.Second)
	}
	s.Require().NoError(err)
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {

			s.T().Skip("Skipping CVE count tests for now.")

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
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearch() {
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(!features.PostgresDatastore.Enabled())
	if !features.PostgresDatastore.Enabled() {
		// Wait for bleve indexing to complete
		time.Sleep(1 * time.Second)
	}
	s.Require().NoError(err)
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {

			testCtx := s.nodeTestContexts[c.contextKey]
			results, err := s.nodeCVEStore.Search(testCtx, nil)
			s.NoError(err)
			expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
			for name, visible := range c.expectedCVEFound {
				if visible {
					expectedCVENames = append(expectedCVENames, s.getNodeCVEID(name))
				}
			}
			fetchedCVEIDs := make(map[string]bool, 0)
			for _, result := range results {
				fetchedCVEIDs[result.ID] = true
			}
			fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
			for id := range fetchedCVEIDs {
				fetchedCVENames = append(fetchedCVENames, id)
			}
			s.ElementsMatch(fetchedCVENames, expectedCVENames)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearchCVEs() {
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(!features.PostgresDatastore.Enabled())
	if !features.PostgresDatastore.Enabled() {
		// Wait for bleve indexing to complete
		time.Sleep(1 * time.Second)
	}
	s.Require().NoError(err)
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {

			testCtx := s.nodeTestContexts[c.contextKey]
			results, err := s.nodeCVEStore.SearchCVEs(testCtx, nil)
			s.NoError(err)
			expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
			for name, visible := range c.expectedCVEFound {
				if visible {
					expectedCVENames = append(expectedCVENames, s.getNodeCVEID(name))
				}
			}
			fetchedCVEIDs := make(map[string]bool, 0)
			for _, result := range results {
				fetchedCVEIDs[result.GetId()] = true
			}
			fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
			for id := range fetchedCVEIDs {
				fetchedCVENames = append(fetchedCVENames, id)
			}
			s.ElementsMatch(fetchedCVENames, expectedCVENames)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearchRawCVEs() {
	err := s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph(!features.PostgresDatastore.Enabled())
	if !features.PostgresDatastore.Enabled() {
		// Wait for bleve indexing to complete
		time.Sleep(1 * time.Second)
	}
	s.Require().NoError(err)
	for _, c := range nodeCVETestCases {
		s.Run(c.contextKey, func() {

			testCtx := s.nodeTestContexts[c.contextKey]
			results, err := s.nodeCVEStore.SearchRawCVEs(testCtx, nil)
			s.NoError(err)
			expectedCVENames := make([]string, 0, len(c.expectedCVEFound))
			for name, visible := range c.expectedCVEFound {
				if visible {
					expectedCVENames = append(expectedCVENames, s.getNodeCVEID(name))
				}
			}
			fetchedCVEIDs := make(map[string]bool, 0)
			for _, result := range results {
				fetchedCVEIDs[result.GetId()] = true
			}
			fetchedCVENames := make([]string, 0, len(fetchedCVEIDs))
			for id := range fetchedCVEIDs {
				fetchedCVENames = append(fetchedCVENames, id)
			}
			s.ElementsMatch(fetchedCVENames, expectedCVENames)
		})
	}
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
