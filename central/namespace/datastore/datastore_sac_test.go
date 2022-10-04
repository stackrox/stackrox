package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/namespace/index/mappings"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestNamespaceDataStoreSAC(t *testing.T) {
	suite.Run(t, new(namespaceDatastoreSACSuite))
}

type namespaceDatastoreSACSuite struct {
	suite.Suite

	// Elements for bleve+rocksdb mode
	engine   *rocksdb.RocksDB
	index    bleve.Index
	dacky    *dackbox.DackBox
	keyFence concurrency.KeyFence
	indexQ   queue.WaitableQueue

	// Elements for postgres mode
	pgtestbase *pgtest.TestPostgres

	datastore DataStore

	optionsMap searchPkg.OptionsMap

	testContexts     map[string]context.Context
	testNamespaceIDs []string
}

func (s *namespaceDatastoreSACSuite) SetupSuite() {
	var err error
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.pgtestbase = pgtest.ForT(s.T())
		s.Require().NotNil(s.pgtestbase)
		s.datastore, err = GetTestPostgresDataStore(s.T(), s.pgtestbase.Pool)
		s.Require().NoError(err)
		s.optionsMap = schema.NamespacesSchema.OptionsMap
	} else {
		s.engine, err = rocksdb.NewTemp("namespaceSACTest")
		s.Require().NoError(err)
		s.index, err = globalindex.MemOnlyIndex()
		s.Require().NoError(err)
		s.keyFence = concurrency.NewKeyFence()
		s.indexQ = queue.NewWaitableQueue()
		s.dacky, err = dackbox.NewRocksDBDackBox(s.engine, s.indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
		s.Require().NoError(err)

		s.datastore, err = GetTestRocksBleveDataStore(s.T(), s.engine, s.index, s.dacky, s.keyFence)
		s.Require().NoError(err)
		s.optionsMap = mappings.OptionsMap
	}

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.Namespace)
}

func (s *namespaceDatastoreSACSuite) TearDownSuite() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.pgtestbase.Pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
		s.Require().NoError(s.index.Close())
	}
}

func (s *namespaceDatastoreSACSuite) SetupTest() {
	s.testNamespaceIDs = make([]string, 0)

	namespaces := fixtures.GetSACTestStorageNamespaceMetadataSet(fixtures.GetScopedNamespace)

	for i := range namespaces {
		err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], namespaces[i])
		s.Require().NoError(err)
	}

	for _, namespace := range namespaces {
		s.testNamespaceIDs = append(s.testNamespaceIDs, namespace.GetId())
	}
}

func (s *namespaceDatastoreSACSuite) TearDownTest() {
	for _, id := range s.testNamespaceIDs {
		s.deleteNamespace(id)
	}
}

func (s *namespaceDatastoreSACSuite) deleteNamespace(id string) {
	s.Require().NoError(s.datastore.RemoveNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *namespaceDatastoreSACSuite) TestAddNamespace() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testutils.VerbAdd)

	for name, c := range cases {
		s.Run(name, func() {
			namespace := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testNamespaceIDs = append(s.testNamespaceIDs, namespace.GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.AddNamespace(ctx, namespace)
			defer s.deleteNamespace(namespace.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *namespaceDatastoreSACSuite) TestGetNamespace() {
	namespace := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	namespace.Priority = 1
	err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], namespace)
	s.Require().NoError(err)
	s.testNamespaceIDs = append(s.testNamespaceIDs, namespace.GetId())

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetNamespace(ctx, namespace.GetId())
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.Require().True(found)
				s.Equal(*namespace, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *namespaceDatastoreSACSuite) TestGetNamespaces() {
	// Remove data injected in SetupTest.
	for _, id := range s.testNamespaceIDs {
		s.deleteNamespace(id)
	}
	s.testNamespaceIDs = s.testNamespaceIDs[:0]

	// Inject data for current test.
	cluster1NamespaceA := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA)
	cluster1NamespaceC := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceC)
	cluster2NamespaceB := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	cluster2NamespaceC := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceC)
	cluster3NamespaceA := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster3, testconsts.NamespaceA)
	testNamespaces := []*storage.NamespaceMetadata{
		cluster1NamespaceA,
		cluster1NamespaceC,
		cluster2NamespaceB,
		cluster2NamespaceC,
		cluster3NamespaceA}
	for _, namespace := range testNamespaces {
		err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], namespace)
		s.Require().NoError(err)
		s.testNamespaceIDs = append(s.testNamespaceIDs, namespace.GetId())
	}

	cases := []struct {
		ScopeKey          string
		VisibleNamespaces []*storage.NamespaceMetadata
	}{
		{
			ScopeKey:          testutils.UnrestrictedReadCtx,
			VisibleNamespaces: testNamespaces,
		},
		{
			ScopeKey:          testutils.UnrestrictedReadWriteCtx,
			VisibleNamespaces: testNamespaces,
		},
		{
			ScopeKey:          testutils.Cluster1ReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster1NamespaceA, cluster1NamespaceC},
		},
		{
			ScopeKey:          testutils.Cluster1NamespaceAReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster1NamespaceA},
		},
		{
			ScopeKey:          testutils.Cluster1NamespaceBReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{},
		},
		{
			ScopeKey:          testutils.Cluster1NamespaceCReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster1NamespaceC},
		},
		{
			ScopeKey:          testutils.Cluster1NamespacesABReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster1NamespaceA},
		},
		{
			ScopeKey:          testutils.Cluster1NamespacesACReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster1NamespaceA, cluster1NamespaceC},
		},
		{
			ScopeKey:          testutils.Cluster1NamespacesBCReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster1NamespaceC},
		},
		{
			ScopeKey:          testutils.Cluster2ReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster2NamespaceB, cluster2NamespaceC},
		},
		{
			ScopeKey:          testutils.Cluster2NamespaceAReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{},
		},
		{
			ScopeKey:          testutils.Cluster2NamespaceBReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster2NamespaceB},
		},
		{
			ScopeKey:          testutils.Cluster2NamespaceCReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster2NamespaceC},
		},
		{
			ScopeKey:          testutils.Cluster2NamespacesABReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster2NamespaceB},
		},
		{
			ScopeKey:          testutils.Cluster2NamespacesACReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster2NamespaceC},
		},
		{
			ScopeKey:          testutils.Cluster2NamespacesBCReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster2NamespaceB, cluster2NamespaceC},
		},
		{
			ScopeKey:          testutils.Cluster3ReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster3NamespaceA},
		},
		{
			ScopeKey:          testutils.Cluster3NamespaceAReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster3NamespaceA},
		},
		{
			ScopeKey:          testutils.Cluster3NamespaceBReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{},
		},
		{
			ScopeKey:          testutils.Cluster3NamespaceCReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{},
		},
		{
			ScopeKey:          testutils.Cluster3NamespacesABReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster3NamespaceA},
		},
		{
			ScopeKey:          testutils.Cluster3NamespacesACReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster3NamespaceA},
		},
		{
			ScopeKey:          testutils.Cluster3NamespacesBCReadWriteCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{},
		},
		{
			ScopeKey:          testutils.MixedClusterAndNamespaceReadCtx,
			VisibleNamespaces: []*storage.NamespaceMetadata{cluster1NamespaceA, cluster2NamespaceB, cluster2NamespaceC},
		},
	}

	for _, c := range cases {
		s.Run(c.ScopeKey, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, err := s.datastore.GetNamespaces(ctx)
			s.Require().NoError(err)
			expectedNamespaceIDs := make([]string, 0, len(c.VisibleNamespaces))
			for ix := range c.VisibleNamespaces {
				expectedNamespaceIDs = append(expectedNamespaceIDs, c.VisibleNamespaces[ix].GetId())
			}
			retrievedNamespaceIDs := make([]string, 0, len(res))
			for ix := range res {
				retrievedNamespaceIDs = append(retrievedNamespaceIDs, res[ix].GetId())
			}
			s.ElementsMatch(retrievedNamespaceIDs, expectedNamespaceIDs)
		})
	}
}

func (s *namespaceDatastoreSACSuite) TestRemoveNamespace() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			namespace := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testNamespaceIDs = append(s.testNamespaceIDs, namespace.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], namespace)
			s.Require().NoError(err)
			defer s.deleteNamespace(namespace.GetId())

			err = s.datastore.RemoveNamespace(ctx, namespace.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *namespaceDatastoreSACSuite) TestUpdateNamespace() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testutils.VerbUpdate)

	for name, c := range cases {
		s.Run(name, func() {
			namespace := fixtures.GetScopedNamespace(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testNamespaceIDs = append(s.testNamespaceIDs, namespace.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.AddNamespace(s.testContexts[testutils.UnrestrictedReadWriteCtx], namespace)
			s.Require().NoError(err)
			defer s.deleteNamespace(namespace.GetId())

			err = s.datastore.UpdateNamespace(ctx, namespace)
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *namespaceDatastoreSACSuite) runCountTest(testparams testutils.SACSearchTestCase) {
	ctx := s.testContexts[testparams.ScopeKey]
	resultCount, err := s.datastore.Count(ctx, nil)
	s.NoError(err)
	expectedResultCount := testutils.AggregateCounts(s.T(), testparams.Results)
	s.Equal(expectedResultCount, resultCount)
}

func (s *namespaceDatastoreSACSuite) TestScopedCount() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runCountTest(c)
		})
	}
}

func (s *namespaceDatastoreSACSuite) TestUnrestrictedCount() {
	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runCountTest(c)
		})
	}
}

func (s *namespaceDatastoreSACSuite) runSearchTest(testparams testutils.SACSearchTestCase) {
	ctx := s.testContexts[testparams.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultCounts := testutils.CountResultsPerClusterAndNamespace(s.T(), results, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testparams.Results, resultCounts)
}

func (s *namespaceDatastoreSACSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *namespaceDatastoreSACSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *namespaceDatastoreSACSuite) runSearchResultsTest(testparams testutils.SACSearchTestCase) {
	ctx := s.testContexts[testparams.ScopeKey]
	results, err := s.datastore.SearchResults(ctx, nil)
	s.Require().NoError(err)
	resultCounts := testutils.CountSearchResultsPerClusterAndNamespace(s.T(), results, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testparams.Results, resultCounts)
}

func (s *namespaceDatastoreSACSuite) TestScopedSearchResults() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchResultsTest(c)
		})
	}
}

func (s *namespaceDatastoreSACSuite) TestUnrestrictedSearchResults() {
	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchResultsTest(c)
		})
	}
}

func (s *namespaceDatastoreSACSuite) countSearchResultObjectsPerClusterAndNamespace(results []*storage.NamespaceMetadata) map[string]map[string]int {
	resultDistribution := make(map[string]map[string]int, 0)
	for _, result := range results {
		if result == nil {
			continue
		}
		clusterID := result.GetClusterId()
		namespace := result.GetName()
		if _, clusterIDExists := resultDistribution[clusterID]; !clusterIDExists {
			resultDistribution[clusterID] = make(map[string]int, 0)
		}
		if _, namespaceExists := resultDistribution[clusterID][namespace]; !namespaceExists {
			resultDistribution[clusterID][namespace] = 0
		}
		resultDistribution[clusterID][namespace]++
	}
	return resultDistribution

}

func (s *namespaceDatastoreSACSuite) runSearchNamespacesTest(testparams testutils.SACSearchTestCase) {
	ctx := s.testContexts[testparams.ScopeKey]
	results, err := s.datastore.SearchNamespaces(ctx, nil)
	s.Require().NoError(err)
	resultCounts := s.countSearchResultObjectsPerClusterAndNamespace(results)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testparams.Results, resultCounts)
}

func (s *namespaceDatastoreSACSuite) TestScopedSearchNamespaces() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchNamespacesTest(c)
		})
	}
}

func (s *namespaceDatastoreSACSuite) TestUnrestrictedSearchNamespaces() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchNamespacesTest(c)
		})
	}
}
