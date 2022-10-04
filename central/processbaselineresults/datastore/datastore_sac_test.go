package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestProcessBaselineResultsDatastoreSAC(t *testing.T) {
	suite.Run(t, new(processBaselineResultsDatastoreSACSuite))
}

type processBaselineResultsDatastoreSACSuite struct {
	suite.Suite

	engine *rocksdb.RocksDB

	pool *pgxpool.Pool

	datastore                  DataStore
	testContexts               map[string]context.Context
	testProcessBaselineResults []string
}

func (s *processBaselineResultsDatastoreSACSuite) SetupSuite() {
	var err error
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		pgtestbase := pgtest.ForT(s.T())
		s.Require().NotNil(pgtestbase)
		s.pool = pgtestbase.Pool
		s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
		s.Require().NoError(err)
	} else {
		s.engine, err = rocksdb.NewTemp("riskSACTest")
		s.Require().NoError(err)

		s.datastore, err = GetTestRocksBleveDataStore(s.T(), s.engine)
		s.Require().NoError(err)
	}

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.ProcessWhitelist)
}

func (s *processBaselineResultsDatastoreSACSuite) TearDownSuite() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.pool.Close()
	} else {
		s.Require().NoError(rocksdb.CloseAndRemove(s.engine))
	}
}

func (s *processBaselineResultsDatastoreSACSuite) SetupTest() {
	s.testProcessBaselineResults = make([]string, 0)

	processBaselineResults := fixtures.GetSACTestStorageProcessBaselineResultsSet(fixtures.GetScopedProcessBaselineResult)

	for i := range processBaselineResults {
		err := s.datastore.UpsertBaselineResults(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaselineResults[i])
		s.Require().NoError(err)
	}

	for _, processBaseline := range processBaselineResults {
		s.testProcessBaselineResults = append(s.testProcessBaselineResults, processBaseline.GetDeploymentId())
	}
}

func (s *processBaselineResultsDatastoreSACSuite) TearDownTest() {
	for _, id := range s.testProcessBaselineResults {
		s.deleteProcessBaselineResult(id)
	}
}

func (s *processBaselineResultsDatastoreSACSuite) deleteProcessBaselineResult(id string) {
	s.Require().NoError(s.datastore.DeleteBaselineResults(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *processBaselineResultsDatastoreSACSuite) TestUpsertBaselineResults() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, c := range cases {
		s.Run(name, func() {
			processBaselineResult := fixtures.GetScopedProcessBaselineResult(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testProcessBaselineResults = append(s.testProcessBaselineResults, processBaselineResult.GetDeploymentId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertBaselineResults(ctx, processBaselineResult)
			defer s.deleteProcessBaselineResult(processBaselineResult.GetDeploymentId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *processBaselineResultsDatastoreSACSuite) TestGetBaselineResults() {
	processBaselineResult := fixtures.GetScopedProcessBaselineResult(uuid.NewV4().String(), testconsts.Cluster2,
		testconsts.NamespaceB)
	err := s.datastore.UpsertBaselineResults(s.testContexts[testutils.UnrestrictedReadWriteCtx], processBaselineResult)
	s.Require().NoError(err)
	s.testProcessBaselineResults = append(s.testProcessBaselineResults, processBaselineResult.GetDeploymentId())

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, err := s.datastore.GetBaselineResults(ctx, processBaselineResult.GetDeploymentId())
			if c.ExpectedFound {
				s.NoError(err)
				s.Equal(*processBaselineResult, *res)
			} else {
				s.Require().Error(err)
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
				s.Nil(res)
			}
		})
	}
}

func (s *processBaselineResultsDatastoreSACSuite) TestDeleteBaselineResults() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			processBaselineResult := fixtures.GetScopedProcessBaselineResult(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			err := s.datastore.UpsertBaselineResults(s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaselineResult)
			s.Require().NoError(err)
			s.testProcessBaselineResults = append(s.testProcessBaselineResults, processBaselineResult.GetDeploymentId())
			defer s.deleteProcessBaselineResult(processBaselineResult.GetDeploymentId())

			ctx := s.testContexts[c.ScopeKey]
			err = s.datastore.DeleteBaselineResults(ctx, processBaselineResult.GetDeploymentId())
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
			} else {
				s.NoError(err)
			}
		})
	}
}
