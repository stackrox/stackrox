package datastore

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store/postgres"
	rdbStore "github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store/rocksdb"
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

func TestProcessBaselineResultsDatastoreSAC(t *testing.T) {
	suite.Run(t, new(processBaselineResultsDatastoreSACSuite))
}

type processBaselineResultsDatastoreSACSuite struct {
	suite.Suite

	engine *rocksdb.RocksDB

	pool *pgxpool.Pool

	storage store.Store

	datastore                  DataStore
	testContexts               map[string]context.Context
	testProcessBaselineResults []string
}

func (s *processBaselineResultsDatastoreSACSuite) SetupSuite() {
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
	} else {
		s.engine, err = rocksdb.NewTemp("riskSACTest")
		s.Require().NoError(err)
		s.storage = rdbStore.New(s.engine)
	}

	s.datastore = New(s.storage)

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(),
		resources.ProcessWhitelist)
}

func (s *processBaselineResultsDatastoreSACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
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
		"read-write on matching cluster and matching namespace should be able to add": {
			scopeKey: testutils.Cluster2NamespaceBReadWriteCtx,
		},
		"read-write on matching cluster and no namespace should be able to add": {
			scopeKey: testutils.Cluster2ReadWriteCtx,
		},
		"read-write on matching cluster and at least one matching namespace should be able to add": {
			scopeKey: testutils.Cluster2NamespacesABReadWriteCtx,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			processBaselineResult := fixtures.GetScopedProcessBaselineResult(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			s.testProcessBaselineResults = append(s.testProcessBaselineResults, processBaselineResult.GetDeploymentId())
			ctx := s.testContexts[c.scopeKey]
			err := s.datastore.UpsertBaselineResults(ctx, processBaselineResult)
			defer s.deleteProcessBaselineResult(processBaselineResult.GetDeploymentId())
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
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

	cases := map[string]struct {
		scopeKey    string
		expectFail  bool
		expectedErr error
	}{
		"global read-only can get": {
			scopeKey: testutils.UnrestrictedReadCtx,
		},
		"global read-write can get": {
			scopeKey: testutils.UnrestrictedReadWriteCtx,
		},
		"read-write on wrong cluster cannot get": {
			scopeKey:    testutils.Cluster1ReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and wrong namespace cannot get": {
			scopeKey:    testutils.Cluster1NamespaceAReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on wrong cluster and matching namespace cannot get": {
			scopeKey:    testutils.Cluster1NamespaceBReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster but wrong namespaces cannot get": {
			scopeKey:    testutils.Cluster2NamespacesACReadWriteCtx,
			expectFail:  true,
			expectedErr: sac.ErrResourceAccessDenied,
		},
		"read-write on matching cluster can read": {
			scopeKey: testutils.Cluster2ReadWriteCtx,
		},
		"read-write on the matching cluster and namespace can get": {
			scopeKey: testutils.Cluster2NamespaceBReadWriteCtx,
		},
		"read-write on the matching cluster and at least one matching namespace can get": {
			scopeKey: testutils.Cluster2NamespacesABReadWriteCtx,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			res, err := s.datastore.GetBaselineResults(ctx, processBaselineResult.GetDeploymentId())
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
				s.Nil(res)
			} else {
				s.NoError(err)
				s.Equal(*processBaselineResult, *res)
			}
		})
	}
}

func (s *processBaselineResultsDatastoreSACSuite) TestDeleteBaselineResults() {
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
		"full read-write on matching cluster can remove": {
			scopeKey: testutils.Cluster2ReadWriteCtx,
		},
		"read-write on the matching cluster and namespace can remove": {
			scopeKey: testutils.Cluster2NamespaceBReadWriteCtx,
		},
		"read-write on the matching cluster and at least the right namespace can remove": {
			scopeKey: testutils.Cluster2NamespacesABReadWriteCtx,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			processBaselineResult := fixtures.GetScopedProcessBaselineResult(uuid.NewV4().String(), testconsts.Cluster2,
				testconsts.NamespaceB)
			err := s.datastore.UpsertBaselineResults(s.testContexts[testutils.UnrestrictedReadWriteCtx],
				processBaselineResult)
			s.Require().NoError(err)
			s.testProcessBaselineResults = append(s.testProcessBaselineResults, processBaselineResult.GetDeploymentId())
			defer s.deleteProcessBaselineResult(processBaselineResult.GetDeploymentId())

			ctx := s.testContexts[c.scopeKey]
			err = s.datastore.DeleteBaselineResults(ctx, processBaselineResult.GetDeploymentId())
			if c.expectFail {
				s.Require().Error(err)
				s.ErrorIs(err, c.expectedErr)
			} else {
				s.NoError(err)
			}
		})
	}
}
