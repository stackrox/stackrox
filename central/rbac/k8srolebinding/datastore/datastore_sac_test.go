package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/mappings"
	"github.com/stackrox/rox/central/role/resources"
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

func TestK8sRoleBindingSAC(t *testing.T) {
	suite.Run(t, new(k8sRoleBindingSACSuite))
}

type k8sRoleBindingSACSuite struct {
	suite.Suite

	datastore DataStore

	pool *pgxpool.Pool

	engine *rocksdb.RocksDB
	index  bleve.Index

	optionsMap searchPkg.OptionsMap

	testContexts          map[string]context.Context
	testK8sRoleBindingIDs []string
}

func (s *k8sRoleBindingSACSuite) SetupSuite() {
	var err error

	if features.PostgresDatastore.Enabled() {
		pgtestbase := pgtest.ForT(s.T())
		s.Require().NotNil(pgtestbase)
		s.pool = pgtestbase.Pool
		s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
		s.Require().NoError(err)
		s.optionsMap = schema.RoleBindingsSchema.OptionsMap
	} else {
		s.engine, err = rocksdb.NewTemp("k8sRoleBindingSACTest")
		s.Require().NoError(err)
		s.index, err = globalindex.MemOnlyIndex()
		s.Require().NoError(err)

		s.datastore, err = GetTestRocksBleveDataStore(s.T(), s.engine, s.index)
		s.Require().NoError(err)
		s.optionsMap = mappings.OptionsMap
	}

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.K8sRoleBinding)
}

func (s *k8sRoleBindingSACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
		s.Require().NoError(s.index.Close())
	}
}

func (s *k8sRoleBindingSACSuite) SetupTest() {
	s.testK8sRoleBindingIDs = make([]string, 0)

	k8sRoleBindings := fixtures.GetSACTestStorageK8SRoleBindingSet(fixtures.GetScopedK8SRoleBinding)

	for i := range k8sRoleBindings {
		err := s.datastore.UpsertRoleBinding(s.testContexts[testutils.UnrestrictedReadWriteCtx], k8sRoleBindings[i])
		s.Require().NoError(err)
	}

	for _, rb := range k8sRoleBindings {
		s.testK8sRoleBindingIDs = append(s.testK8sRoleBindingIDs, rb.GetId())
	}
}

func (s *k8sRoleBindingSACSuite) TearDownTest() {
	for _, id := range s.testK8sRoleBindingIDs {
		s.deleteK8sRoleBinding(id)
	}
}

func (s *k8sRoleBindingSACSuite) deleteK8sRoleBinding(id string) {
	s.Require().NoError(s.datastore.RemoveRoleBinding(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *k8sRoleBindingSACSuite) TestUpsertRoleBinding() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, c := range cases {
		s.Run(name, func() {
			roleBinding := fixtures.GetScopedK8SRoleBinding(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testK8sRoleBindingIDs = append(s.testK8sRoleBindingIDs, roleBinding.GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertRoleBinding(ctx, roleBinding)
			defer s.deleteK8sRoleBinding(roleBinding.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *k8sRoleBindingSACSuite) TestGetRoleBinding() {
	roleBinding := fixtures.GetScopedK8SRoleBinding(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.UpsertRoleBinding(s.testContexts[testutils.UnrestrictedReadWriteCtx], roleBinding)
	s.Require().NoError(err)
	s.testK8sRoleBindingIDs = append(s.testK8sRoleBindingIDs, roleBinding.GetId())

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetRoleBinding(ctx, roleBinding.GetId())
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Equal(*roleBinding, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *k8sRoleBindingSACSuite) TestRemoveRoleBinding() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			roleBinding := fixtures.GetScopedK8SRoleBinding(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testK8sRoleBindingIDs = append(s.testK8sRoleBindingIDs, roleBinding.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertRoleBinding(s.testContexts[testutils.UnrestrictedReadWriteCtx], roleBinding)
			s.Require().NoError(err)
			defer s.deleteK8sRoleBinding(roleBinding.GetId())

			err = s.datastore.RemoveRoleBinding(ctx, roleBinding.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *k8sRoleBindingSACSuite) runSearchRawTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawRoleBindings(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i])
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *k8sRoleBindingSACSuite) runSearchTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultCounts := testutils.CountResultsPerClusterAndNamespace(s.T(), results, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *k8sRoleBindingSACSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *k8sRoleBindingSACSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *k8sRoleBindingSACSuite) TestScopeSearchRaw() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *k8sRoleBindingSACSuite) TestUnrestrictedSearchRaw() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}
