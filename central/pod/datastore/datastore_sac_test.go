package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/pod/mappings"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestPodDatastoreSAC(t *testing.T) {
	suite.Run(t, new(podDatastoreSACSuite))
}

type podDatastoreSACSuite struct {
	suite.Suite

	datastore DataStore

	pool *pgxpool.Pool

	engine *rocksdb.RocksDB
	index  bleve.Index

	testContexts map[string]context.Context
	testPodIDs   []string
}

func (s *podDatastoreSACSuite) SetupSuite() {
	var err error

	if features.PostgresDatastore.Enabled() {
		pgtestbase := pgtest.ForT(s.T())
		s.Require().NotNil(pgtestbase)
		s.pool = pgtestbase.Pool
		s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
		s.Require().NoError(err)
	} else {
		s.engine, err = rocksdb.NewTemp("podSACTest")
		s.Require().NoError(err)
		s.index, err = globalindex.MemOnlyIndex()
		s.Require().NoError(err)

		s.datastore, err = GetTestRocksBleveDataStore(s.T(), s.engine, s.index)
		s.Require().NoError(err)
	}

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.Deployment)
}

func (s *podDatastoreSACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
		s.Require().NoError(s.index.Close())
	}
}

func (s *podDatastoreSACSuite) SetupTest() {
	s.testPodIDs = make([]string, 0)

	pods := fixtures.GetSACTestStoragePodSet(fixtures.GetScopedPod)
	for i := range pods {
		s.Require().NoError(s.datastore.UpsertPod(s.testContexts[testutils.UnrestrictedReadWriteCtx], pods[i]))
	}

	for _, p := range pods {
		s.testPodIDs = append(s.testPodIDs, p.GetId())
	}
}

func (s *podDatastoreSACSuite) TearDownTest() {
	for _, id := range s.testPodIDs {
		s.deletePod(id)
	}
}

func (s *podDatastoreSACSuite) deletePod(id string) {
	s.Require().NoError(s.datastore.RemovePod(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *podDatastoreSACSuite) TestUpsertPod() {
	testedVerb := "upsert"
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testedVerb)

	for name, c := range cases {
		s.Run(name, func() {
			pod := fixtures.GetScopedPod(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testPodIDs = append(s.testPodIDs, pod.GetId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertPod(ctx, pod)
			defer s.deletePod(pod.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *podDatastoreSACSuite) TestGetPod() {
	pod := fixtures.GetScopedPod(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err := s.datastore.UpsertPod(s.testContexts[testutils.UnrestrictedReadWriteCtx], pod)
	s.Require().NoError(err)
	s.testPodIDs = append(s.testPodIDs, pod.GetId())

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetPod(ctx, pod.GetId())
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Equal(*pod, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *podDatastoreSACSuite) TestRemovePod() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			pod := fixtures.GetScopedPod(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testPodIDs = append(s.testPodIDs, pod.GetId())

			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertPod(s.testContexts[testutils.UnrestrictedReadWriteCtx], pod)
			s.Require().NoError(err)
			defer s.deletePod(pod.GetId())

			err = s.datastore.RemovePod(ctx, pod.GetId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *podDatastoreSACSuite) runSearchRawTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.SearchRawPods(ctx, nil)
	s.Require().NoError(err)
	resultObjs := make([]sac.NamespaceScopedObject, 0, len(results))
	for i := range results {
		resultObjs = append(resultObjs, results[i])
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), resultObjs)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *podDatastoreSACSuite) runSearchTest(c testutils.SACSearchTestCase) {
	ctx := s.testContexts[c.ScopeKey]
	results, err := s.datastore.Search(ctx, nil)
	s.Require().NoError(err)
	resultCounts := testutils.CountResultsPerClusterAndNamespace(s.T(), results, mappings.OptionsMap)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, c.Results, resultCounts)
}

func (s *podDatastoreSACSuite) TestScopedSearch() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *podDatastoreSACSuite) TestUnrestrictedSearch() {
	for name, c := range testutils.GenericUnrestrictedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchTest(c)
		})
	}
}

func (s *podDatastoreSACSuite) TestScopedSearchRaw() {
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}

func (s *podDatastoreSACSuite) TestUnrestrictedSearchRaw() {
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runSearchRawTest(c)
		})
	}
}
