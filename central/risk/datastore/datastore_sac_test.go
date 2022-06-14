package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/risk/datastore/internal/store/postgres"
	rdbStore "github.com/stackrox/rox/central/risk/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/risk/mappings"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
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

func TestRiskDataStoreSAC(t *testing.T) {
	suite.Run(t, new(riskDatastoreSACSuite))
}

type riskDatastoreSACSuite struct {
	suite.Suite

	engine *rocksdb.RocksDB
	index  bleve.Index

	pool *pgxpool.Pool

	storage store.Store
	indexer index.Indexer
	search  search.Searcher

	datastore DataStore

	optionsMap searchPkg.OptionsMap

	testContexts map[string]context.Context
	testRiskIDs  []string
}

func (s *riskDatastoreSACSuite) SetupSuite() {
	var err error
	if features.PostgresDatastore.Enabled() {
		ctx := context.Background()
		src := pgtest.GetConnectionString(s.T())
		cfg, err := pgxpool.ParseConfig(src)
		s.Require().NoError(err)
		s.pool, err = pgxpool.ConnectConfig(ctx, cfg)
		s.Require().NoError(err)
		pgStore.Destroy(ctx, s.pool)
		gormDB := pgtest.OpenGormDB(s.T(), src)
		defer pgtest.CloseGormDB(s.T(), gormDB)
		s.storage = pgStore.CreateTableAndNewStore(ctx, s.pool, gormDB)
		s.indexer = pgStore.NewIndexer(s.pool)
		s.optionsMap = schema.RisksSchema.OptionsMap
	} else {
		s.engine, err = rocksdb.NewTemp("riskSACTest")
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
		resources.Risk)
}

func (s *riskDatastoreSACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
		s.Require().NoError(s.index.Close())
	}
}

func (s *riskDatastoreSACSuite) SetupTest() {
	s.testRiskIDs = make([]string, 0)

	risks := fixtures.GetSACTestStorageRiskSet(fixtures.GetScopedRisk)

	for i := range risks {
		err := s.datastore.UpsertRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], risks[i])
		s.Require().NoError(err)
	}

	for _, risk := range risks {
		s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())
	}
}

func (s *riskDatastoreSACSuite) TearDownTest() {
	for _, id := range s.testRiskIDs {
		s.deleteRisk(id)
	}
}

func (s *riskDatastoreSACSuite) deleteRisk(id string) {
	s.Require().NoError(s.datastore.RemoveRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], id,
		storage.RiskSubjectType_DEPLOYMENT))
}

func (s *riskDatastoreSACSuite) TestUpsertRisk() {
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
			risk := fixtures.GetScopedRisk(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())
			ctx := s.testContexts[c.scopeKey]
			err := s.datastore.UpsertRisk(ctx, risk)
			defer s.deleteRisk(risk.GetSubject().GetId())
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *riskDatastoreSACSuite) TestGetRisk() {
	risk := fixtures.GetScopedRisk(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.UpsertRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], risk)
	s.Require().NoError(err)
	s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())

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
		"read-write on matching cluster cannot read": {
			scopeKey: testutils.Cluster2ReadWriteCtx,
		},
		"read-write on the matching cluster and namespace cannot get": {
			scopeKey: testutils.Cluster2NamespaceBReadWriteCtx,
		},
		"read-write on the matching cluster and at least one matching namespace cannot get": {
			scopeKey: testutils.Cluster2NamespacesABReadWriteCtx,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			res, found, err := s.datastore.GetRisk(ctx, risk.GetSubject().GetId(), storage.RiskSubjectType_DEPLOYMENT)
			s.Require().NoError(err)
			if c.found {
				s.Require().True(found)
				s.Equal(*risk, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *riskDatastoreSACSuite) TestGetRiskForDeployment() {
	risk := fixtures.GetScopedRisk(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.UpsertRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], risk)
	s.Require().NoError(err)
	s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())

	d := &storage.Deployment{
		Id:        risk.GetSubject().GetId(),
		ClusterId: testconsts.Cluster2,
		Namespace: testconsts.NamespaceB,
	}

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
			res, found, err := s.datastore.GetRiskForDeployment(ctx, d)
			s.Require().NoError(err)
			if c.found {
				s.Require().True(found)
				s.Equal(*risk, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *riskDatastoreSACSuite) TestRemoveRisk() {
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
			risk := fixtures.GetScopedRisk(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testRiskIDs = append(s.testRiskIDs, risk.GetSubject().GetId())

			ctx := s.testContexts[c.scopeKey]
			err := s.datastore.UpsertRisk(s.testContexts[testutils.UnrestrictedReadWriteCtx], risk)
			s.Require().NoError(err)
			defer s.deleteRisk(risk.GetId())

			err = s.datastore.RemoveRisk(ctx, risk.GetSubject().GetId(), storage.RiskSubjectType_DEPLOYMENT)
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *riskDatastoreSACSuite) runSearchRawTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawRisks(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i].Subject)
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *riskDatastoreSACSuite) runSearchTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultCounts := testutils.CountResultsPerClusterAndNamespace(s.T(), results, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *riskDatastoreSACSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *riskDatastoreSACSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *riskDatastoreSACSuite) TestScopeSearchRaw() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *riskDatastoreSACSuite) TestUnrestrictedSearchRaw() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}
