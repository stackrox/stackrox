package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	pgStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	rdbStore "github.com/stackrox/rox/central/processindicator/store/rocksdb"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	sacTestUtils "github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	mappings "github.com/stackrox/rox/pkg/search/options/processindicators"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestProcessIndicatorDataStoreSAC(t *testing.T) {
	suite.Run(t, new(processIndicatorDatastoreSACSuite))
}

type processIndicatorDatastoreSACSuite struct {
	suite.Suite

	engine *rocksdb.RocksDB
	index  bleve.Index

	pool *pgxpool.Pool

	storage store.Store
	indexer index.Indexer
	search  search.Searcher

	datastore DataStore

	optionsMap searchPkg.OptionsMap

	testContexts            map[string]context.Context
	testProcessIndicatorIDs []string
}

func (s *processIndicatorDatastoreSACSuite) SetupSuite() {
	var err error
	processIndicatorObj := "processIndicatorSACTest"

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
		s.optionsMap = schema.ProcessIndicatorsSchema.OptionsMap
	} else {
		s.engine, err = rocksdb.NewTemp(processIndicatorObj)
		s.Require().NoError(err)
		bleveIndex, err := globalindex.MemOnlyIndex()
		s.Require().NoError(err)
		s.index = bleveIndex

		s.storage = rdbStore.New(s.engine)
		s.indexer = index.New(s.index)
		s.optionsMap = mappings.OptionsMap
	}

	s.search = search.New(s.storage, s.indexer)
	s.datastore, err = New(s.storage, s.indexer, s.search, nil)
	s.Require().NoError(err)

	s.testContexts = sacTestUtils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.Indicator)
}

func (s *processIndicatorDatastoreSACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		err := rocksdb.CloseAndRemove(s.engine)
		s.Require().NoError(err)
		s.Require().NoError(s.index.Close())
	}

}

func (s *processIndicatorDatastoreSACSuite) SetupTest() {
	s.testProcessIndicatorIDs = make([]string, 0)

	processIndicators := fixtures.GetSACTestStorageProcessIndicatorSet(fixtures.GetScopedProcessIndicator)
	err := s.datastore.AddProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], processIndicators...)
	s.Require().NoError(err)

	for _, pi := range processIndicators {
		s.testProcessIndicatorIDs = append(s.testProcessIndicatorIDs, pi.GetId())
	}
}

func (s *processIndicatorDatastoreSACSuite) TearDownTest() {
	err := s.datastore.RemoveProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx],
		s.testProcessIndicatorIDs)
	s.Require().NoError(err)
}

func (s *processIndicatorDatastoreSACSuite) deleteProcessIndicator(id string) {
	s.Require().NoError(s.datastore.RemoveProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx],
		[]string{id}))
}

func (s *processIndicatorDatastoreSACSuite) TestAddProcessIndicators() {
	cases := map[string]struct {
		scopeKey    string
		expectFail  bool
		expectedErr error
	}{
		"global read-only should not be able to add": {
			scopeKey:    sacTestUtils.UnrestrictedReadCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"global read-write should be able to add": {
			scopeKey: sacTestUtils.UnrestrictedReadWriteCtx,
		},
		"read-write on wrong cluster should not be able to add": {
			scopeKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and namespace should not be able to add": {
			scopeKey:    sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace should not be able to add": {
			scopeKey:    sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and wrong namespace should not be able to add": {
			scopeKey:    sacTestUtils.Cluster2NamespaceAReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and matching namespace should be able to add": {
			scopeKey:    sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster and at least one matching namespace should be able to add": {
			scopeKey:    sacTestUtils.Cluster2NamespacesABReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			processIndicator := fixtures.GetScopedProcessIndicator(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testProcessIndicatorIDs = append(s.testProcessIndicatorIDs, processIndicator.GetId())
			ctx := s.testContexts[c.scopeKey]
			err := s.datastore.AddProcessIndicators(ctx, processIndicator)
			defer s.deleteProcessIndicator(processIndicator.GetId())
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}

}

func (s *processIndicatorDatastoreSACSuite) TestGetProcessIndicator() {
	processIndicator := fixtures.GetScopedProcessIndicator(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.AddProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], processIndicator)
	s.Require().NoError(err)
	s.testProcessIndicatorIDs = append(s.testProcessIndicatorIDs, processIndicator.GetId())

	cases := map[string]struct {
		scopeKey string
		found    bool
	}{
		"global read-only can get": {
			scopeKey: sacTestUtils.UnrestrictedReadCtx,
			found:    true,
		},
		"global read-write can get": {
			scopeKey: sacTestUtils.UnrestrictedReadWriteCtx,
			found:    true,
		},
		"read-write on wrong cluster cannot get": {
			scopeKey: sacTestUtils.Cluster1ReadWriteCtx,
		},
		"read-write on wrong cluster and wrong namespace cannot get": {
			scopeKey: sacTestUtils.Cluster1NamespaceAReadWriteCtx,
		},
		"read-write on wrong cluster and matching namespace cannot get": {
			scopeKey: sacTestUtils.Cluster1NamespaceBReadWriteCtx,
		},
		"read-write on matching cluster but wrong namespaces cannot get": {
			scopeKey: sacTestUtils.Cluster2NamespacesACReadWriteCtx,
		},
		"read-write on matching cluster can read": {
			scopeKey: sacTestUtils.Cluster2ReadWriteCtx,
			found:    true,
		},
		"read-write on the matching cluster and namespace can get": {
			scopeKey: sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			found:    true,
		},
		"read-write on the matching cluster and at least one matching namespace can get": {
			scopeKey: sacTestUtils.Cluster2NamespacesABReadWriteCtx,
			found:    true,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			res, found, err := s.datastore.GetProcessIndicator(ctx, processIndicator.GetId())
			s.Require().NoError(err)
			if c.found {
				s.True(found)
				s.Equal(*processIndicator, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestRemoveProcessIndicators() {
	cases := map[string]struct {
		scopeKey    string
		expectFail  bool
		expectedErr error
	}{
		"global read-only cannot remove": {
			scopeKey:    sacTestUtils.UnrestrictedReadCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"global read-write can remove": {
			scopeKey:    sacTestUtils.UnrestrictedReadWriteCtx,
			expectedErr: nil,
		},
		"read-write on wrong cluster cannot remove": {
			scopeKey:    sacTestUtils.Cluster1ReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and wrong namespace cannot remove": {
			scopeKey:    sacTestUtils.Cluster1NamespaceAReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace cannot remove": {
			scopeKey:    sacTestUtils.Cluster1NamespaceBReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster but wrong namespaces cannot remove": {
			scopeKey:    sacTestUtils.Cluster2NamespacesACReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"full read-write on matching cluster cannot remove": {
			scopeKey:    sacTestUtils.Cluster2ReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on the matching cluster and namespace cannot remove": {
			scopeKey:    sacTestUtils.Cluster2NamespaceBReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on the matching cluster and at least the right namespace cannot remove": {
			scopeKey:    sacTestUtils.Cluster2NamespacesABReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			processIndicator := fixtures.GetScopedProcessIndicator(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testProcessIndicatorIDs = append(s.testProcessIndicatorIDs, processIndicator.GetId())

			ctx := s.testContexts[c.scopeKey]
			err := s.datastore.AddProcessIndicators(s.testContexts[sacTestUtils.UnrestrictedReadWriteCtx], processIndicator)
			s.Require().NoError(err)
			defer s.deleteProcessIndicator(processIndicator.GetId())

			err = s.datastore.RemoveProcessIndicators(ctx, []string{processIndicator.GetId()})
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestScopedSearch() {
	for name, c := range sacTestUtils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestUnrestrictedSearch() {
	for name, c := range sacTestUtils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestScopeSearchRaw() {
	for name, c := range sacTestUtils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) TestUnrestrictedSearchRaw() {
	for name, c := range sacTestUtils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *processIndicatorDatastoreSACSuite) runSearchRawTest(c sacTestUtils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawProcessIndicators(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i])
	}
	resultCounts := sacTestUtils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	sacTestUtils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *processIndicatorDatastoreSACSuite) runSearchTest(c sacTestUtils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultCounts := sacTestUtils.CountResultsPerClusterAndNamespace(s.T(), results, s.optionsMap)
	sacTestUtils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}
