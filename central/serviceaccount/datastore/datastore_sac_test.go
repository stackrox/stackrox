package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/serviceaccount/mappings"
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

	optionsMap searchPkg.OptionsMap

	testContexts          map[string]context.Context
	testServiceAccountIDs []string
}

func (s *serviceAccountSACSuite) SetupSuite() {
	var err error

	if features.PostgresDatastore.Enabled() {
		pgtestbase := pgtest.ForT(s.T())
		s.Require().NotNil(pgtestbase)
		s.pool = pgtestbase.Pool
		s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
		s.Require().NoError(err)
		s.optionsMap = schema.ServiceAccountsSchema.OptionsMap
	} else {
		s.engine, err = rocksdb.NewTemp("serviceAccountSACTest")
		s.Require().NoError(err)
		s.index, err = globalindex.MemOnlyIndex()
		s.Require().NoError(err)
		s.datastore, err = GetTestRocksBleveDataStore(s.T(), s.engine, s.index)
		s.Require().NoError(err)
		s.optionsMap = mappings.OptionsMap
	}

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
	testedVerb := "upsert"
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testedVerb)

	for name, c := range cases {
		s.Run(name, func() {
			account := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, account.GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertServiceAccount(ctx, account)
			defer s.deleteServiceAccount(account.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
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

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetServiceAccount(ctx, account.GetId())
			s.Require().NoError(err)
			if c.ExpectedFound {
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
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			account := fixtures.GetScopedServiceAccount(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testServiceAccountIDs = append(s.testServiceAccountIDs, account.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertServiceAccount(s.testContexts[testutils.UnrestrictedReadWriteCtx], account)
			s.Require().NoError(err)
			defer s.deleteServiceAccount(account.GetId())

			err = s.datastore.RemoveServiceAccount(ctx, account.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
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
