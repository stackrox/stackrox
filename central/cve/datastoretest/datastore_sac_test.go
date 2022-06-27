package datastoretest

import (
	"context"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/cve/converter"
	genericCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	dackboxTestUtils "github.com/stackrox/rox/central/dackbox/testutils"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
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

type imageCVEDataStoreFromGenericStore struct {
	genericStore genericCVEDataStore.DataStore
}

func isImageCVE(genericCVE *storage.CVE) bool {
	if genericCVE.GetType() == storage.CVE_IMAGE_CVE {
		return true
	}
	for _, cveType := range genericCVE.GetTypes() {
		if cveType == storage.CVE_IMAGE_CVE {
			return true
		}
	}
	return false
}

func (s *imageCVEDataStoreFromGenericStore) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return s.genericStore.Search(ctx, q)
}

func (s *imageCVEDataStoreFromGenericStore) SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return s.genericStore.SearchCVEs(ctx, q)
}

func (s *imageCVEDataStoreFromGenericStore) SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVE, error) {
	cves, error := s.genericStore.SearchRawCVEs(ctx, q)
	if error != nil {
		return nil, error
	}
	imageCVES := make([]*storage.ImageCVE, 0, len(cves))
	for ix := range cves {
		cve := cves[ix]
		if !isImageCVE(cve) {
			continue
		}
		imageCVES = append(imageCVES, converter.ProtoCVEToImageCVE(cve))
	}
	return imageCVES, nil
}

func (s *imageCVEDataStoreFromGenericStore) Exists(ctx context.Context, id string) (bool, error) {
	return s.genericStore.Exists(ctx, id)
}

func (s *imageCVEDataStoreFromGenericStore) Get(ctx context.Context, id string) (*storage.ImageCVE, bool, error) {
	cve, found, err := s.genericStore.Get(ctx, id)
	if err != nil || !found {
		return nil, found, err
	}
	if !isImageCVE(cve) {
		return nil, false, nil
	}
	return converter.ProtoCVEToImageCVE(cve), true, nil
}

func (s *imageCVEDataStoreFromGenericStore) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.genericStore.Count(ctx, q)
}

func (s *imageCVEDataStoreFromGenericStore) GetBatch(ctx context.Context, id []string) ([]*storage.ImageCVE, error) {
	cves, err := s.genericStore.GetBatch(ctx, id)
	if err != nil {
		return nil, err
	}
	imageCVEs := make([]*storage.ImageCVE, 0, len(cves))
	for _, cve := range cves {
		if !isImageCVE(cve) {
			continue
		}
		imageCVEs = append(imageCVEs, converter.ProtoCVEToImageCVE(cve))
	}
	return imageCVEs, nil
}

func (s *imageCVEDataStoreFromGenericStore) Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, cves ...string) error {
	return s.genericStore.Suppress(ctx, start, duration, cves...)
}

func (s *imageCVEDataStoreFromGenericStore) Unsuppress(ctx context.Context, cves ...string) error {
	return s.genericStore.Unsuppress(ctx, cves...)
}

func (s *imageCVEDataStoreFromGenericStore) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	s.genericStore.EnrichImageWithSuppressedCVEs(image)
}

type nodeCVEDataStoreFromGenericStore struct {
	genericStore genericCVEDataStore.DataStore
}

func isNodeCVE(genericCVE *storage.CVE) bool {
	if genericCVE.GetType() == storage.CVE_NODE_CVE {
		return true
	}
	for _, cveType := range genericCVE.GetTypes() {
		if cveType == storage.CVE_NODE_CVE {
			return true
		}
	}
	return false
}

func (s *nodeCVEDataStoreFromGenericStore) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return s.genericStore.Search(ctx, q)
}

func (s *nodeCVEDataStoreFromGenericStore) SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return s.genericStore.SearchCVEs(ctx, q)
}

func (s *nodeCVEDataStoreFromGenericStore) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.NodeCVE, error) {
	cves, err := s.genericStore.SearchRawCVEs(ctx, q)
	if err != nil {
		return nil, err
	}
	nodeCVEs := make([]*storage.NodeCVE, 0, len(cves))
	for _, cve := range cves {
		fmt.Println("CVE ", cve.GetId(), "(", cve.GetCve(), ")", isNodeCVE(cve))
		if !isNodeCVE(cve) {
			continue
		}
		nodeCVEs = append(nodeCVEs, converter.ProtoCVEToNodeCVE(cve))
	}
	return nodeCVEs, nil
}

func (s *nodeCVEDataStoreFromGenericStore) Exists(ctx context.Context, id string) (bool, error) {
	return s.genericStore.Exists(ctx, id)
}

func (s *nodeCVEDataStoreFromGenericStore) Get(ctx context.Context, id string) (*storage.NodeCVE, bool, error) {
	cve, found, err := s.genericStore.Get(ctx, id)
	if err != nil || !found {
		return nil, found, err
	}
	if !isNodeCVE(cve) {
		return nil, false, nil
	}
	return converter.ProtoCVEToNodeCVE(cve), true, nil
}

func (s *nodeCVEDataStoreFromGenericStore) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.genericStore.Count(ctx, q)
}

func (s *nodeCVEDataStoreFromGenericStore) GetBatch(ctx context.Context, id []string) ([]*storage.NodeCVE, error) {
	cves, err := s.genericStore.GetBatch(ctx, id)
	if err != nil {
		return nil, err
	}
	nodeCVEs := make([]*storage.NodeCVE, 0, len(cves))
	for _, cve := range cves {
		if !isNodeCVE(cve) {
			continue
		}
		nodeCVEs = append(nodeCVEs, converter.ProtoCVEToNodeCVE(cve))
	}
	return nodeCVEs, nil
}

func (s *nodeCVEDataStoreFromGenericStore) Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, cves ...string) error {
	return s.genericStore.Suppress(ctx, start, duration, cves...)
}

func (s *nodeCVEDataStoreFromGenericStore) Unsuppress(ctx context.Context, cves ...string) error {
	return s.genericStore.Unsuppress(ctx, cves...)
}

func (s *nodeCVEDataStoreFromGenericStore) EnrichNodeWithSuppressedCVEs(node *storage.Node) {
	s.genericStore.EnrichNodeWithSuppressedCVEs(node)
}

func (s *cveDataStoreSACTestSuite) SetupSuite() {
	var err error
	s.dackboxTestStore, err = dackboxTestUtils.NewDackboxTestDataStore(s.T())
	s.Require().NoError(err)
	if features.PostgresDatastore.Enabled() {
		ctx := context.Background()
		pool := s.dackboxTestStore.GetPostgresPool()
		src := pgtest.GetConnectionString(s.T())
		gormDB := pgtest.OpenGormDB(s.T(), src)
		defer pgtest.CloseGormDB(s.T(), gormDB)
		s.imageCVEStore, err = imageCVEDataStore.GetTestPostgresDataStore(ctx, s.T(), pool, gormDB)
		s.Require().NoError(err)
		s.nodeCVEStore, err = nodeCVEDataStore.GetTestPostgresDataStore(ctx, s.T(), pool, gormDB)
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
	s.Require().NoError(s.dackboxTestStore.Cleanup())
}

func (s *cveDataStoreSACTestSuite) getCVEID(cve string) string {
	if features.PostgresDatastore.Enabled() {
		return cve + "#Linux"
	}
	return cve
}

func (s *cveDataStoreSACTestSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.dackboxTestStore.CleanImageToVulnerabilitiesGraph())
}

func (s *cveDataStoreSACTestSuite) cleanNodeToVulnerabilitiesGraph() {
	s.Require().NoError(s.dackboxTestStore.CleanNodeToVulnerabilitiesGraph())
}

type existsTestCase struct {
	contextKey     string
	expectedExists bool
}

type getTestCase struct {
	contextKey    string
	expectedFound bool
}

type readMultiTestCase struct {
	contextKey      string
	expectedFetched []string
}

type countTestCase struct {
	contextKey    string
	expectedCount int
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsSingleScopeOnly() {
	// Inject the fixture graph, and test exists for CVE-1234-0001
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	cveId := s.getCVEID("CVE-1234-0001")
	testCases := map[string]existsTestCase{
		"Unrestricted read-write can see single scope CVE": {
			contextKey:     sacTestUtils.UnrestrictedReadWriteCtx,
			expectedExists: true,
		},
		"Unrestricted read can see single scope CVE": {
			contextKey:     sacTestUtils.UnrestrictedReadCtx,
			expectedExists: true,
		},
		"Right cluster full read-write can see single scope CVE": {
			contextKey:     sacTestUtils.Cluster1ReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write to right namespace can see single scope CVE": {
			contextKey:     sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write to wrong namespace can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
		},
		"Right cluster partial read-write to at least the right namespace can see single scope CVE": {
			contextKey:     sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedExists: true,
		},
		"Wrong cluster full read-write can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
		},
		"Wrong cluster partial read-write to the matching namespace can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster2NamespaceAReadWriteCtx,
		},
		"Wrong cluster partial read-write to any not-matching namespace can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
		},
	}
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.imageTestContexts[c.contextKey]
			exists, err := s.imageCVEStore.Exists(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedExists, exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsSharedAcrossComponents() {
	// Inject the fixture graph, and test exists for CVE-4567-0002
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	cveId := s.getCVEID("CVE-4567-0002")
	testCases := map[string]existsTestCase{
		"Unrestricted read-write can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.UnrestrictedReadWriteCtx,
			expectedExists: true,
		},
		"Unrestricted read can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.UnrestrictedReadCtx,
			expectedExists: true,
		},
		"Right cluster full read-write can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.Cluster1ReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write to right namespace can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write to wrong namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
		},
		"Right cluster partial read-write to at least the right namespace can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedExists: true,
		},
		"Other right cluster full read-write can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.Cluster2ReadWriteCtx,
			expectedExists: true,
		},
		"Other right cluster partial read-write to right namespace can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedExists: true,
		},
		"Other right cluster partial read-write to any wrong namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
		},
		"Other right cluster partial read-write to at least the right namespace can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedExists: true,
		},
		"Wrong cluster full read-write can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
		"Wrong cluster partial read-write to some matching namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
		},
		"Wrong cluster partial read-write to some other matching namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
		},
		"Wrong cluster partial read-write to any matching namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
		},
	}
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.imageTestContexts[c.contextKey]
			exists, err := s.imageCVEStore.Exists(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedExists, exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEExistsFromSharedComponent() {
	// Inject the fixture graph, and test exists for CVE-3456-0004
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	cveId := s.getCVEID("CVE-3456-0004")
	testCases := map[string]existsTestCase{
		"Unrestricted read-write can see CVE from shared components": {
			contextKey:     sacTestUtils.UnrestrictedReadWriteCtx,
			expectedExists: true,
		},
		"Unrestricted read can see CVE from shared components": {
			contextKey:     sacTestUtils.UnrestrictedReadCtx,
			expectedExists: true,
		},
		"Right cluster full read-write can see CVE from shared components": {
			contextKey:     sacTestUtils.Cluster1ReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write to right namespace can see CVE from shared components": {
			contextKey:     sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write to wrong namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
		},
		"Right cluster partial read-write to at least the right namespace can see CVE from shared components": {
			contextKey:     sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedExists: true,
		},
		"Other right cluster full read-write can see CVE from shared components": {
			contextKey:     sacTestUtils.Cluster2ReadWriteCtx,
			expectedExists: true,
		},
		"Other right cluster partial read-write to right namespace can see CVE from shared components": {
			contextKey:     sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedExists: true,
		},
		"Other right cluster partial read-write to any wrong namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
		},
		"Other right cluster partial read-write to at least the right namespace can see CVE from shared components": {
			contextKey:     sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedExists: true,
		},
		"Wrong cluster full read-write can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
		"Wrong cluster partial read-write to some matching namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
		},
		"Wrong cluster partial read-write to some other matching namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
		},
		"Wrong cluster partial read-write to any matching namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
		},
	}
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.imageTestContexts[c.contextKey]
			exists, err := s.imageCVEStore.Exists(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedExists, exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	testCases := map[string]getTestCase{
		"Unrestricted read-write can see single scope CVE": {
			contextKey:    sacTestUtils.UnrestrictedReadWriteCtx,
			expectedFound: true,
		},
		"Unrestricted read can see single scope CVE": {
			contextKey:    sacTestUtils.UnrestrictedReadCtx,
			expectedFound: true,
		},
		"Right cluster full read-write can see single scope CVE": {
			contextKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write to right namespace can see single scope CVE": {
			contextKey:    sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write to wrong namespace can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
		},
		"Right cluster partial read-write to at least the right namespace can see single scope CVE": {
			contextKey:    sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedFound: true,
		},
		"Wrong cluster full read-write can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
		},
		"Wrong cluster partial read-write to the matching namespace can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster2NamespaceAReadWriteCtx,
		},
		"Wrong cluster partial read-write to any not-matching namespace can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
		},
	}
	cve := "CVE-1234-0001"
	cveId := s.getCVEID(cve)
	cvss := float32(5.8)
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.imageTestContexts[c.contextKey]
			imageCVE, found, err := s.imageCVEStore.Get(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedFound, found)
			if c.expectedFound {
				s.Require().NotNil(imageCVE)
				s.Equal(cve, imageCVE.GetCveBaseInfo().GetCve())
				s.Equal(cvss, imageCVE.Cvss)
			} else {
				s.Nil(imageCVE)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetSharedAcrossComponents() {
	// Inject the fixture graph, and test retrieval for CVE-4567-0002
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	testCases := map[string]getTestCase{
		"Unrestricted read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.UnrestrictedReadWriteCtx,
			expectedFound: true,
		},
		"Unrestricted read can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.UnrestrictedReadCtx,
			expectedFound: true,
		},
		"Right cluster full read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write to right namespace can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write to wrong namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
		},
		"Right cluster partial read-write to at least the right namespace can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedFound: true,
		},
		"Other right cluster full read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster2ReadWriteCtx,
			expectedFound: true,
		},
		"Other right cluster partial read-write to right namespace can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedFound: true,
		},
		"Other right cluster partial read-write to any wrong namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
		},
		"Other right cluster partial read-write to at least the right namespace can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedFound: true,
		},
		"Wrong cluster full read-write can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
		"Wrong cluster partial read-write to some matching namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
		},
		"Wrong cluster partial read-write to some other matching namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
		},
		"Wrong cluster partial read-write to any matching namespace can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
		},
	}
	cve := "CVE-4567-0002"
	cveId := s.getCVEID(cve)
	cvss := float32(7.5)
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.imageTestContexts[c.contextKey]
			imageCVE, found, err := s.imageCVEStore.Get(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedFound, found)
			if c.expectedFound {
				s.Require().NotNil(imageCVE)
				s.Equal(cve, imageCVE.GetCveBaseInfo().GetCve())
				s.Equal(cvss, imageCVE.Cvss)
			} else {
				s.Nil(imageCVE)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetFromSharedComponent() {
	// Inject the fixture graph, and test retrieval for CVE-3456-0004
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	testCases := map[string]getTestCase{
		"Unrestricted read-write can see CVE from shared components": {
			contextKey:    sacTestUtils.UnrestrictedReadWriteCtx,
			expectedFound: true,
		},
		"Unrestricted read can see CVE from shared components": {
			contextKey:    sacTestUtils.UnrestrictedReadCtx,
			expectedFound: true,
		},
		"Right cluster full read-write can see CVE from shared components": {
			contextKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write to right namespace can see CVE from shared components": {
			contextKey:    sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write to wrong namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
		},
		"Right cluster partial read-write to at least the right namespace can see CVE from shared components": {
			contextKey:    sacTestUtils.Cluster1NamespacesABReadWriteCtx,
			expectedFound: true,
		},
		"Other right cluster full read-write can see CVE from shared components": {
			contextKey:    sacTestUtils.Cluster2ReadWriteCtx,
			expectedFound: true,
		},
		"Other right cluster partial read-write to right namespace can see CVE from shared components": {
			contextKey:    sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectedFound: true,
		},
		"Other right cluster partial read-write to any wrong namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
		},
		"Other right cluster partial read-write to at least the right namespace can see CVE from shared components": {
			contextKey:    sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedFound: true,
		},
		"Wrong cluster full read-write can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
		"Wrong cluster partial read-write to some matching namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3NamespaceAReadWriteCtx,
		},
		"Wrong cluster partial read-write to some other matching namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3NamespaceBReadWriteCtx,
		},
		"Wrong cluster partial read-write to any matching namespace can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3NamespacesABReadWriteCtx,
		},
	}
	cve := "CVE-3456-0004"
	cveId := s.getCVEID(cve)
	cvss := float32(7.5)
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.imageTestContexts[c.contextKey]
			imageCVE, found, err := s.imageCVEStore.Get(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedFound, found)
			if c.expectedFound {
				s.Require().NotNil(imageCVE)
				s.Equal(cve, imageCVE.GetCveBaseInfo().GetCve())
				s.Equal(cvss, imageCVE.Cvss)
			} else {
				s.Nil(imageCVE)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVEGetBatch() {
	// In the read batch:
	// 0001 (node 1 only),
	// 0002 (nodes 1 and 2),
	// 0003 (node 1 only)
	// 0004 (nodes 1 and 2),
	// 0006 (node 2 only)
	s.dackboxTestStore.PushImageToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	cve1 := "CVE-1234-0001"
	cve2 := "CVE-4567-0002"
	cve3 := "CVE-1234-0003"
	cve4 := "CVE-3456-0004"
	cve6 := "CVE-2345-0006"
	batchCVEs := []string{
		cve1,
		cve2,
		cve3,
		cve4,
		cve6,
	}
	cveIDs := make([]string, 0, len(batchCVEs))
	for _, cve := range batchCVEs {
		cveIDs = append(cveIDs, s.getCVEID(cve))
	}
	testCases := map[string]readMultiTestCase{
		"Unrestricted read-write can see all CVE from request": {
			contextKey:      sacTestUtils.UnrestrictedReadWriteCtx,
			expectedFetched: cveIDs,
		},
		"Unrestricted read can see all CVE from request": {
			contextKey:      sacTestUtils.UnrestrictedReadCtx,
			expectedFetched: cveIDs,
		},
		"Right cluster full read-write can see CVE from request that are linked to the cluster": {
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedFetched: []string{
				s.getCVEID(cve1),
				s.getCVEID(cve2),
				s.getCVEID(cve3),
				s.getCVEID(cve4),
			},
		},
		"Right cluster partial read-write to right namespace can see CVE from request linked to the cluster and namespace": {
			contextKey: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectedFetched: []string{
				s.getCVEID(cve1),
				s.getCVEID(cve2),
				s.getCVEID(cve3),
				s.getCVEID(cve4),
			},
		},
		"Right cluster partial read-write to wrong namespaces can NOT see any CVE from request": {
			contextKey: sacTestUtils.Cluster1NamespacesBCReadWriteCtx,
		},
		"Other right cluster full read-write can see CVE from request that are linked to the cluster": {
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedFetched: []string{
				s.getCVEID(cve2),
				s.getCVEID(cve4),
				s.getCVEID(cve6),
			},
		},
		"Other right cluster partial read-write to at least a right namespace can see CVE from request linked to the valid cluster and namespace": {
			contextKey: sacTestUtils.Cluster2NamespacesBCReadWriteCtx,
			expectedFetched: []string{
				s.getCVEID(cve2),
				s.getCVEID(cve4),
				s.getCVEID(cve6),
			},
		},
		"Other right cluster partial read-write can NOT see any CVE from request": {
			contextKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
		},
		"Wrong cluster can NOT see any CVE from request": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
	}
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.imageTestContexts[c.contextKey]
			nodeCVEs, err := s.imageCVEStore.GetBatch(testCtx, cveIDs)
			s.NoError(err)
			fetchedCVEIDs := make([]string, 0, len(nodeCVEs))
			for _, nodeCVE := range nodeCVEs {
				fetchedCVEIDs = append(fetchedCVEIDs, nodeCVE.GetId())
			}
			s.ElementsMatch(c.expectedFetched, fetchedCVEIDs)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVECount() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearch() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearchCVEs() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACImageCVESearchRawCVEs() {
	s.T().Skip("Not implemented yet.")
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
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	cveId := s.getCVEID("CVE-1234-0001")
	testCases := map[string]existsTestCase{
		"Unrestricted read-write can see single scope CVE": {
			contextKey:     sacTestUtils.UnrestrictedReadWriteCtx,
			expectedExists: true,
		},
		"Unrestricted read can see single scope CVE": {
			contextKey:     sacTestUtils.UnrestrictedReadCtx,
			expectedExists: true,
		},
		"Right cluster full read-write can see single scope CVE": {
			contextKey:     sacTestUtils.Cluster1ReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
		},
		"Wrong cluster full read-write can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
		},
	}
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			exists, err := s.nodeCVEStore.Exists(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedExists, exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsSharedAcrossComponents() {
	// Inject the fixture graph, and test exists for CVE-4567-0002
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	cveId := s.getCVEID("CVE-4567-0002")
	testCases := map[string]existsTestCase{
		"Unrestricted read-write can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.UnrestrictedReadWriteCtx,
			expectedExists: true,
		},
		"Unrestricted read can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.UnrestrictedReadCtx,
			expectedExists: true,
		},
		"Right cluster full read-write can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.Cluster1ReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
		},
		"Other right cluster full read-write can see shared CVE from not shared components": {
			contextKey:     sacTestUtils.Cluster2ReadWriteCtx,
			expectedExists: true,
		},
		"Other right cluster partial read-write can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
		},
		"Wrong cluster can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
	}
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			exists, err := s.nodeCVEStore.Exists(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedExists, exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEExistsFromSharedComponent() {
	// Inject the fixture graph, and test exists for CVE-3456-0004
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	testCases := map[string]existsTestCase{
		"Unrestricted read-write can see CVE from shared components": {
			contextKey:     sacTestUtils.UnrestrictedReadWriteCtx,
			expectedExists: true,
		},
		"Unrestricted read can see CVE from shared components": {
			contextKey:     sacTestUtils.UnrestrictedReadCtx,
			expectedExists: true,
		},
		"Right cluster full read-write can see CVE from shared components": {
			contextKey:     sacTestUtils.Cluster1ReadWriteCtx,
			expectedExists: true,
		},
		"Right cluster partial read-write can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
		},
		"Other right cluster full read-write can see CVE from shared components": {
			contextKey:     sacTestUtils.Cluster2ReadWriteCtx,
			expectedExists: true,
		},
		"Other right cluster partial read-write can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
		},
		"Wrong cluster can NOT see CVE from shared components": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
	}
	cveId := s.getCVEID("CVE-3456-0004")
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			exists, err := s.nodeCVEStore.Exists(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedExists, exists)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetSingleScopeOnly() {
	// Inject the fixture graph, and test retrieval for CVE-1234-0001
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	testCases := map[string]getTestCase{
		"Unrestricted read-write can see single scope CVE": {
			contextKey:    sacTestUtils.UnrestrictedReadWriteCtx,
			expectedFound: true,
		},
		"Unrestricted read can see single scope CVE": {
			contextKey:    sacTestUtils.UnrestrictedReadCtx,
			expectedFound: true,
		},
		"Right cluster full read-write can see single scope CVE": {
			contextKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
		},
		"Wrong cluster full read-write can NOT see single scope CVE": {
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
		},
	}
	cve := "CVE-1234-0001"
	cveId := s.getCVEID(cve)
	cvss := float32(5.8)
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeCVE, found, err := s.nodeCVEStore.Get(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedFound, found)
			if c.expectedFound {
				s.NotNil(nodeCVE)
				s.Equal(cve, nodeCVE.GetCveBaseInfo().GetCve())
				s.Equal(cvss, nodeCVE.Cvss)
			} else {
				s.Nil(nodeCVE)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetSharedAcrossComponents() {
	// Inject the fixture graph, and test retrieval for CVE-4567-0002
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	testCases := map[string]getTestCase{
		"Unrestricted read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.UnrestrictedReadWriteCtx,
			expectedFound: true,
		},
		"Unrestricted read can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.UnrestrictedReadCtx,
			expectedFound: true,
		},
		"Right cluster full read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
		},
		"Other right cluster full read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster2ReadWriteCtx,
			expectedFound: true,
		},
		"Other right cluster partial read-write can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
		},
		"Wrong cluster can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
	}
	cve := "CVE-4567-0002"
	cveId := s.getCVEID(cve)
	cvss := float32(7.5)
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeCVE, found, err := s.nodeCVEStore.Get(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedFound, found)
			if c.expectedFound {
				s.NotNil(nodeCVE)
				s.Equal(cve, nodeCVE.GetCveBaseInfo().GetCve())
				s.Equal(cvss, nodeCVE.Cvss)
			} else {
				s.Nil(nodeCVE)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetFromSharedComponent() {
	// Inject the fixture graph, and test retrieval for CVE-3456-0004
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	testCases := map[string]getTestCase{
		"Unrestricted read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.UnrestrictedReadWriteCtx,
			expectedFound: true,
		},
		"Unrestricted read can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.UnrestrictedReadCtx,
			expectedFound: true,
		},
		"Right cluster full read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectedFound: true,
		},
		"Right cluster partial read-write can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
		},
		"Other right cluster full read-write can see shared CVE from not shared components": {
			contextKey:    sacTestUtils.Cluster2ReadWriteCtx,
			expectedFound: true,
		},
		"Other right cluster partial read-write can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
		},
		"Wrong cluster can NOT see shared CVE from not shared components": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
	}
	cve := "CVE-3456-0004"
	cveId := s.getCVEID(cve)
	cvss := float32(7.5)
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeCVE, found, err := s.nodeCVEStore.Get(testCtx, cveId)
			s.NoError(err)
			s.Equal(c.expectedFound, found)
			if c.expectedFound {
				s.NotNil(nodeCVE)
				s.Equal(cve, nodeCVE.GetCveBaseInfo().GetCve())
				s.Equal(cvss, nodeCVE.Cvss)
			} else {
				s.Nil(nodeCVE)
			}
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVEGetBatch() {
	// In the read batch:
	// 0001 (node 1 only),
	// 0002 (nodes 1 and 2),
	// 0003 (node 1 only)
	// 0004 (nodes 1 and 2),
	// 0006 (node 2 only)
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	cve1 := "CVE-1234-0001"
	cve2 := "CVE-4567-0002"
	cve3 := "CVE-1234-0003"
	cve4 := "CVE-3456-0004"
	cve6 := "CVE-2345-0006"
	batchCVEs := []string{
		cve1,
		cve2,
		cve3,
		cve4,
		cve6,
	}
	cveIDs := make([]string, 0, len(batchCVEs))
	for _, cve := range batchCVEs {
		cveIDs = append(cveIDs, s.getCVEID(cve))
	}
	testCases := map[string]readMultiTestCase{
		"Unrestricted read-write can see all CVE from request": {
			contextKey:      sacTestUtils.UnrestrictedReadWriteCtx,
			expectedFetched: cveIDs,
		},
		"Unrestricted read can see all CVE from request": {
			contextKey:      sacTestUtils.UnrestrictedReadCtx,
			expectedFetched: cveIDs,
		},
		"Right cluster full read-write can see CVE from request that are linked to the cluster": {
			contextKey: sacTestUtils.Cluster1ReadWriteCtx,
			expectedFetched: []string{
				s.getCVEID(cve1),
				s.getCVEID(cve2),
				s.getCVEID(cve3),
				s.getCVEID(cve4),
			},
		},
		"Right cluster partial read-write can NOT see any CVE from request": {
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
		},
		"Other right cluster full read-write can see CVE from request that are linked to the cluster": {
			contextKey: sacTestUtils.Cluster2ReadWriteCtx,
			expectedFetched: []string{
				s.getCVEID(cve2),
				s.getCVEID(cve4),
				s.getCVEID(cve6),
			},
		},
		"Other right cluster partial read-write can NOT see any CVE from request": {
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
		},
		"Wrong cluster can NOT see any CVE from request": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
	}
	for name, c := range testCases {
		s.Run(name, func() {
			if features.PostgresDatastore.Enabled() {
				s.T().Skip("Skipping CVE tests for postgres for now.")
			}
			testCtx := s.nodeTestContexts[c.contextKey]
			nodeCVEs, err := s.nodeCVEStore.GetBatch(testCtx, cveIDs)
			s.NoError(err)
			fetchedCVEIDs := make([]string, 0, len(nodeCVEs))
			for _, nodeCVE := range nodeCVEs {
				fetchedCVEIDs = append(fetchedCVEIDs, nodeCVE.GetId())
			}
			s.ElementsMatch(c.expectedFetched, fetchedCVEIDs)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVECount() {
	s.dackboxTestStore.PushNodeToVulnerabilitiesGraph()
	defer s.cleanNodeToVulnerabilitiesGraph()
	testCases := map[string]countTestCase{
		"Unrestricted read-write can see all CVEs": {
			contextKey:    sacTestUtils.UnrestrictedReadWriteCtx,
			expectedCount: 7,
		},
		"Unrestricted read can see all CVEs": {
			contextKey:    sacTestUtils.UnrestrictedReadCtx,
			expectedCount: 7,
		},
		"Right cluster full read-write can see CVEs that are linked to the cluster": {
			contextKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectedCount: 5,
		},
		"Right cluster partial read-write can NOT see any CVE": {
			contextKey: sacTestUtils.Cluster1NamespacesABReadWriteCtx,
		},
		"Other right cluster full read-write can see CVEs that are linked to the cluster": {
			contextKey:    sacTestUtils.Cluster2ReadWriteCtx,
			expectedCount: 5,
		},
		"Other right cluster partial read-write can NOT see any CVE": {
			contextKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
		},
		"Wrong cluster can NOT see any CVE": {
			contextKey: sacTestUtils.Cluster3ReadWriteCtx,
		},
	}
	for name, c := range testCases {
		s.Run(name, func() {

			s.T().Skip("Skipping CVE count tests for now.")

			testCtx := s.nodeTestContexts[c.contextKey]
			count, err := s.nodeCVEStore.Count(testCtx, nil)
			s.NoError(err)
			s.Equal(c.expectedCount, count)
		})
	}
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearch() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearchCVEs() {
	s.T().Skip("Not implemented yet.")
}

func (s *cveDataStoreSACTestSuite) TestSACNodeCVESearchRawCVEs() {
	s.T().Skip("Not implemented yet.")
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
