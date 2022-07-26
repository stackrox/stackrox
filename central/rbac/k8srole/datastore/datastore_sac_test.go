package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	k8sRoleMappings "github.com/stackrox/rox/central/rbac/k8srole/mappings"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestK8sRoleSAC(t *testing.T) {
	suite.Run(t, new(k8sRoleSACSuite))
}

type k8sRoleSACSuite struct {
	suite.Suite

	datastore DataStore

	pool *pgxpool.Pool

	engine     *rocksdb.RocksDB
	index      bleve.Index
	optionsMap search.OptionsMap

	testContexts   map[string]context.Context
	testK8sRoleIDs []string
}

func (s *k8sRoleSACSuite) SetupSuite() {
	var err error

	if features.PostgresDatastore.Enabled() {
		pgtestbase := pgtest.ForT(s.T())
		s.Require().NotNil(pgtestbase)
		s.pool = pgtestbase.Pool
		s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
		s.Require().NoError(err)
		s.optionsMap = schema.K8sRolesSchema.OptionsMap
	} else {
		s.engine, err = rocksdb.NewTemp("k8sRoleSACTest")
		s.Require().NoError(err)
		s.index, err = globalindex.MemOnlyIndex()
		s.Require().NoError(err)

		s.datastore, err = GetTestRocksBleveDataStore(s.T(), s.engine, s.index)
		s.Require().NoError(err)
		s.optionsMap = k8sRoleMappings.OptionsMap
	}

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.K8sRole)
}

func (s *k8sRoleSACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
		s.Require().NoError(s.index.Close())
	}
}

func (s *k8sRoleSACSuite) SetupTest() {
	s.testK8sRoleIDs = make([]string, 0)

	k8sRoles := fixtures.GetSACTestStorageK8SRoleSet(fixtures.GetScopedK8SRole)

	for i := range k8sRoles {
		err := s.datastore.UpsertRole(s.testContexts[testutils.UnrestrictedReadWriteCtx], k8sRoles[i])
		s.Require().NoError(err)
	}

	for _, rb := range k8sRoles {
		s.testK8sRoleIDs = append(s.testK8sRoleIDs, rb.GetId())
	}
}

func (s *k8sRoleSACSuite) TearDownTest() {
	for _, id := range s.testK8sRoleIDs {
		s.deleteK8sRole(id)
	}
}

func (s *k8sRoleSACSuite) deleteK8sRole(id string) {
	s.Require().NoError(s.datastore.RemoveRole(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *k8sRoleSACSuite) TestUpsertRole() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, c := range cases {
		s.Run(name, func() {
			role := fixtures.GetScopedK8SRole(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testK8sRoleIDs = append(s.testK8sRoleIDs, role.GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertRole(ctx, role)
			defer s.deleteK8sRole(role.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *k8sRoleSACSuite) TestGetRole() {
	role := fixtures.GetScopedK8SRole(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.UpsertRole(s.testContexts[testutils.UnrestrictedReadWriteCtx], role)
	s.Require().NoError(err)
	s.testK8sRoleIDs = append(s.testK8sRoleIDs, role.GetId())

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetRole(ctx, role.GetId())
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Equal(*role, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *k8sRoleSACSuite) TestRemoveRole() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			role := fixtures.GetScopedK8SRole(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testK8sRoleIDs = append(s.testK8sRoleIDs, role.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertRole(s.testContexts[testutils.UnrestrictedReadWriteCtx], role)
			s.Require().NoError(err)
			defer s.deleteK8sRole(role.GetId())

			err = s.datastore.RemoveRole(ctx, role.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *k8sRoleSACSuite) runSearchRawTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawRoles(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i])
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *k8sRoleSACSuite) runSearchTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultCounts := testutils.CountResultsPerClusterAndNamespace(s.T(), results, s.optionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *k8sRoleSACSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *k8sRoleSACSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *k8sRoleSACSuite) TestScopeSearchRaw() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *k8sRoleSACSuite) TestUnrestrictedSearchRaw() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}
