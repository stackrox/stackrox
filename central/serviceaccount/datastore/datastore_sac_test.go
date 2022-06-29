package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/serviceaccount/internal/index"
	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	pgStore "github.com/stackrox/rox/central/serviceaccount/internal/store/postgres"
	rdbStore "github.com/stackrox/rox/central/serviceaccount/internal/store/rocksdb"
	"github.com/stackrox/rox/central/serviceaccount/mappings"
	"github.com/stackrox/rox/central/serviceaccount/search"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestServiceAccountSAC(t *testing.T) {
	suite.Run(t, new(serviceAccountSACSuite))
}

type serviceAccountSACSuite struct {
	suite.Suite

	datastore DataStore

	pool *pgxpool.Pool

	engine *rocksdb.RocksDB
	index  bleve.Index

	storage    store.Store
	indexer    index.Indexer
	search     search.Searcher
	optionsMap searchPkg.OptionsMap

	testContexts          map[string]context.Context
	testServiceAccountIDs []string
}

func (s *serviceAccountSACSuite) SetupSuite() {
	var err error

	if features.PostgresDatastore.Enabled() {
		ctx := context.Background()
		src := pgtest.GetConnectionString(s.T())
		cfg, err := pgxpool.ParseConfig(src)
		s.Require().NoError(err)
		s.pool, err = pgxpool.ConnectConfig(ctx, cfg)
		s.Require().NoError(err)
		pgStore.Destroy(ctx, s.pool)
		gormDB := pgtest.OpenGormDB(s.T(), src, false)
		defer pgtest.CloseGormDB(s.T(), gormDB)
		s.storage = pgStore.CreateTableAndNewStore(ctx, s.pool, gormDB)
		s.indexer = pgStore.NewIndexer(s.pool)
		s.optionsMap = schema.ServiceAccountsSchema.OptionsMap
	} else {
		s.engine, err = rocksdb.NewTemp("serviceAccountSACTest")
		s.Require().NoError(err)
		bleveIndex, err := globalindex.MemOnlyIndex()
		s.Require().NoError(err)
		s.index = bleveIndex

		s.storage = rdbStore.New(s.engine)
		s.indexer = index.New(s.index)
		s.optionsMap = mappings.OptionsMap
	}

	s.search = search.New(s.storage, s.indexer)
	s.datastore, err = New(s.storage, s.indexer, s.search)
	s.Require().NoError(err)

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.ServiceAccount)
}

func (s *serviceAccountSACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
		s.Require().NoError(s.index.Close())
	}
}

func (s *serviceAccountSACSuite) SetupTest() {
	s.testServiceAccountIDs = make([]string, 0)

	serviceAccounts := fixtures.GetSACTestStorageServiceAccountSet(fixtures.GetScopedServiceAccount)

	for i := range serviceAccounts {
		err := s.datastore.UpsertServiceAccount(s.testContexts[testutils.UnrestrictedReadWriteCtx], serviceAccounts[i])
		s.Require().NoError(err)
	}

	for _, rb := range serviceAccounts {
		s.testServiceAccountIDs = append(s.testServiceAccountIDs, rb.GetId())
	}
}

func (s *serviceAccountSACSuite) TearDownTest() {
	for _, id := range s.testServiceAccountIDs {
		s.deleteServiceAccount(id)
	}
}

func (s *serviceAccountSACSuite) deleteServiceAccount(id string) {
	s.Require().NoError(s.datastore.RemoveServiceAccount(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *serviceAccountSACSuite) TestUpsertServiceAccount() {
	cases := map[string]struct {
		scopeKey    string
		expectFail  bool
		expectedErr error
	}{
		"global read-only should not be able to add": {
			scopeKey:    testutils.UnrestrictedReadCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"global read-write should be able to add": {
			scopeKey: testutils.UnrestrictedReadWriteCtx,
		},
		"read-write on wrong cluster should not be able to add": {
			scopeKey:    testutils.Cluster1ReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and namespace should not be able to add": {
			scopeKey:    testutils.Cluster1NamespaceAReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace should not be able to add": {
			scopeKey:    testutils.Cluster1NamespaceBReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and wrong namespace should not be able to add": {
			scopeKey:    testutils.Cluster2NamespaceAReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and no namespace should not be able to add": {
			scopeKey:    testutils.Cluster2ReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			account := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, account.GetId())
			ctx := s.testContexts[c.scopeKey]
			err := s.datastore.UpsertServiceAccount(ctx, account)
			defer s.deleteServiceAccount(account.GetId())
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *serviceAccountSACSuite) TestGetServiceAccount() {
	account := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.UpsertServiceAccount(s.testContexts[testutils.UnrestrictedReadWriteCtx], account)
	s.Require().NoError(err)
	s.testServiceAccountIDs = append(s.testServiceAccountIDs, account.GetId())

	cases := map[string]struct {
		scopeKey string
		found    bool
	}{
		"global read-only can get": {
			scopeKey: testutils.UnrestrictedReadCtx,
			found:    true,
		},
		"global read-write can get": {
			scopeKey: testutils.UnrestrictedReadWriteCtx,
			found:    true,
		},
		"read-write on wrong cluster cannot get": {
			scopeKey: testutils.Cluster1ReadWriteCtx,
		},
		"read-write on wrong cluster and wrong namespace cannot get": {
			scopeKey: testutils.Cluster1NamespaceAReadWriteCtx,
		},
		"read-write on wrong cluster and matching namespace cannot get": {
			scopeKey: testutils.Cluster1NamespaceBReadWriteCtx,
		},
		"read-write on matching cluster but wrong namespaces cannot get": {
			scopeKey: testutils.Cluster2NamespacesACReadWriteCtx,
		},
		"read-write on matching cluster can read": {
			scopeKey: testutils.Cluster2ReadWriteCtx,
			found:    true,
		},
		"read-write on the matching cluster and namespace can get": {
			scopeKey: testutils.Cluster2NamespaceBReadWriteCtx,
			found:    true,
		},
		"read-write on the matching cluster and at least one matching namespace can get": {
			scopeKey: testutils.Cluster2NamespacesABReadWriteCtx,
			found:    true,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			res, found, err := s.datastore.GetServiceAccount(ctx, account.GetId())
			s.Require().NoError(err)
			if c.found {
				s.True(found)
				s.Equal(*account, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *serviceAccountSACSuite) TestRemoveServiceAccount() {
	cases := map[string]struct {
		scopeKey    string
		expectFail  bool
		expectedErr error
	}{
		"global read-only cannot remove": {
			scopeKey:    testutils.UnrestrictedReadCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"global read-write can remove": {
			scopeKey:    testutils.UnrestrictedReadWriteCtx,
			expectedErr: nil,
		},
		"read-write on wrong cluster cannot remove": {
			scopeKey:    testutils.Cluster1ReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and wrong namespace cannot remove": {
			scopeKey:    testutils.Cluster1NamespaceAReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace cannot remove": {
			scopeKey:    testutils.Cluster1NamespaceBReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster but wrong namespaces cannot remove": {
			scopeKey:    testutils.Cluster2NamespacesACReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"full read-write on matching cluster cannot remove": {
			scopeKey:    testutils.Cluster2ReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on the matching cluster and namespace cannot remove": {
			scopeKey:    testutils.Cluster2NamespaceBReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on the matching cluster and at least the right namespace cannot remove": {
			scopeKey:    testutils.Cluster2NamespacesABReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			account := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, account.GetId())

			ctx := s.testContexts[c.scopeKey]
			err := s.datastore.UpsertServiceAccount(s.testContexts[testutils.UnrestrictedReadWriteCtx], account)
			s.Require().NoError(err)
			defer s.deleteServiceAccount(account.GetId())

			err = s.datastore.RemoveServiceAccount(ctx, account.GetId())
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *serviceAccountSACSuite) TestSearchServiceAccount() {
	// Run both scoped and unrestricted search test cases.
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchServiceAccountTest(c)
		})
	}

	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchServiceAccountTest(c)
		})
	}
}

func (s *serviceAccountSACSuite) runSearchServiceAccountTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchServiceAccounts(ctx, nil)
	s.Require().NoError(err)
	resultCounts := testutils.CountSearchResultsPerClusterAndNamespace(s.T(), results, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)

}

func (s *serviceAccountSACSuite) runSearchRawTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawServiceAccounts(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i])
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *serviceAccountSACSuite) runSearchTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultCounts := testutils.CountResultsPerClusterAndNamespace(s.T(), results, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *serviceAccountSACSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *serviceAccountSACSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *serviceAccountSACSuite) TestScopeSearchRaw() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *serviceAccountSACSuite) TestUnrestrictedSearchRaw() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}
