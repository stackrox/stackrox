//go:build sql_integration

package datastore

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	cleanupCtxKey = testutils.UnrestrictedReadWriteCtx
)

func TestNetworkBaselineDatastoreSAC(t *testing.T) {
	suite.Run(t, new(networkBaselineDatastoreSACTestSuite))
}

type networkBaselineDatastoreSACTestSuite struct {
	suite.Suite
	pool         postgres.DB
	datastore    DataStore
	testContexts map[string]context.Context
	testNBIDs    []string
}

var _ interface {
	suite.SetupAllSuite
	suite.TearDownAllSuite
	suite.SetupTestSuite
	suite.TearDownTestSuite
} = (*networkBaselineDatastoreSACTestSuite)(nil)

func (s *networkBaselineDatastoreSACTestSuite) SetupSuite() {
	var err error

	pgtestbase := pgtest.ForT(s.T())
	s.Require().NotNil(pgtestbase)
	s.pool = pgtestbase.DB
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.pool)
	s.Require().NoError(err)

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.DeploymentExtension)
}

func (s *networkBaselineDatastoreSACTestSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *networkBaselineDatastoreSACTestSuite) SetupTest() {
	s.testNBIDs = make([]string, 0)
}

func (s *networkBaselineDatastoreSACTestSuite) TearDownTest() {
	for _, id := range s.testNBIDs {
		s.cleanupNetworkBaseline(id)
	}
}

type crudTest struct {
	scopeKey    string
	expectFound bool
}

func (s *networkBaselineDatastoreSACTestSuite) cleanupNetworkBaseline(ID string) {
	err := s.datastore.DeleteNetworkBaseline(s.testContexts[cleanupCtxKey], ID)
	s.NoError(err)
}

func (s *networkBaselineDatastoreSACTestSuite) TestGetNetworkBaseline() {
	var err error
	testNB := fixtures.GetScopedNetworkBaseline(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err = s.datastore.UpsertNetworkBaselines(s.testContexts[testutils.UnrestrictedReadWriteCtx], []*storage.NetworkBaseline{testNB})
	s.testNBIDs = append(s.testNBIDs, testNB.GetDeploymentId())
	s.NoError(err)

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			readNetworkBaseline, found, getErr := s.datastore.GetNetworkBaseline(ctx, testNB.GetDeploymentId())
			s.NoError(getErr)
			s.Equal(c.ExpectedFound, found)
			if c.ExpectedFound {
				s.Equal(testNB, readNetworkBaseline)
			} else {
				s.Nil(readNetworkBaseline)
			}
		})
	}
}

func (s *networkBaselineDatastoreSACTestSuite) TestWalkNetworkBaseline() {
	var err error
	testNB := fixtures.GetScopedNetworkBaseline(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err = s.datastore.UpsertNetworkBaselines(s.testContexts[testutils.UnrestrictedReadWriteCtx], []*storage.NetworkBaseline{testNB})
	s.testNBIDs = append(s.testNBIDs, testNB.GetDeploymentId())
	s.NoError(err)

	cases := map[string]crudTest{
		"(full) read-only can walk": {
			scopeKey:    testutils.UnrestrictedReadCtx,
			expectFound: true,
		},
		"full read-write can walk": {
			scopeKey:    testutils.UnrestrictedReadCtx,
			expectFound: true,
		},
		"full read-write on wrong cluster cannot walk": {
			scopeKey:    testutils.Cluster1ReadWriteCtx,
			expectFound: false,
		},
		"read-write on wrong cluster and wrong namespace cannot walk": {
			scopeKey:    testutils.Cluster1NamespaceAReadWriteCtx,
			expectFound: false,
		},
		"read-write on wrong cluster and matching namespace cannot walk": {
			scopeKey:    testutils.Cluster1NamespaceBReadWriteCtx,
			expectFound: false,
		},
		"read-write on right cluster but wrong namespaces cannot walk": {
			scopeKey:    testutils.Cluster2NamespacesACReadWriteCtx,
			expectFound: false,
		},
		"full read-write on right cluster cannot walk": {
			scopeKey:    testutils.Cluster2ReadWriteCtx,
			expectFound: false,
		},
		"read-write on the right cluster and namespace cannot walk": {
			scopeKey:    testutils.Cluster2NamespaceBReadWriteCtx,
			expectFound: false,
		},
		"read-write on the right cluster and at least the right namespace cannot walk": {
			scopeKey:    testutils.Cluster2NamespacesABReadWriteCtx,
			expectFound: false,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			var found []string
			err := s.datastore.Walk(ctx, func(baseline *storage.NetworkBaseline) error {
				found = append(found, baseline.GetDeploymentId())
				if !c.expectFound {
					return errors.New(baseline.GetDeploymentId())
				}
				return nil
			})
			s.NoError(err)
			if c.expectFound {
				s.ElementsMatch([]string{testNB.GetDeploymentId()}, found)
			} else {
				s.Empty(found)
			}
		})
	}
}

func (s *networkBaselineDatastoreSACTestSuite) TestUpsertNetworkBaselines() {
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, c := range cases {
		s.Run(name, func() {
			testNB := fixtures.GetScopedNetworkBaseline(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testNBIDs = append(s.testNBIDs, testNB.GetDeploymentId())
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertNetworkBaselines(ctx, []*storage.NetworkBaseline{testNB})
			s.Equal(c.ExpectedError, err)

			_, ok, err := s.datastore.GetNetworkBaseline(ctx, testNB.GetDeploymentId())
			s.NoError(err)
			s.Equal(c.ExpectedError == nil, ok, "The resource must exist if Upsert succeeded, or not otherwise")
		})
	}
}

func (s *networkBaselineDatastoreSACTestSuite) TestDeleteNetworkBaseline() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			testNB := fixtures.GetScopedNetworkBaseline(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testNBIDs = append(s.testNBIDs, testNB.GetDeploymentId())
			s.NoError(s.datastore.UpsertNetworkBaselines(s.testContexts[testutils.UnrestrictedReadWriteCtx], []*storage.NetworkBaseline{testNB}))
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.DeleteNetworkBaseline(ctx, testNB.GetDeploymentId())
			s.Equal(c.ExpectedError, err)
			_, ok, err := s.datastore.GetNetworkBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], testNB.GetDeploymentId())
			s.NoError(err)
			s.Equal(c.ExpectedError != nil, ok, "The resource must still exist if Delete failed, or not otherwise")
		})
	}
}

func (s *networkBaselineDatastoreSACTestSuite) TestDeleteNetworkBaselines() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			testNB := fixtures.GetScopedNetworkBaseline(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			s.testNBIDs = append(s.testNBIDs, testNB.GetDeploymentId())
			ctx := s.testContexts[c.ScopeKey]
			var err error
			err = s.datastore.UpsertNetworkBaselines(s.testContexts[testutils.UnrestrictedReadWriteCtx], []*storage.NetworkBaseline{testNB})
			defer s.cleanupNetworkBaseline(testNB.GetDeploymentId())
			s.NoError(err)
			err = s.datastore.DeleteNetworkBaselines(ctx, []string{testNB.GetDeploymentId()})
			s.Equal(c.ExpectedError, err)
		})
	}

	s.Run("Delete multiple NetworkBaselines", func() {
		var nbs = fixtures.GetSACTestNetworkBaseline()
		// Upsert resources with various scopes.
		s.NoError(s.datastore.UpsertNetworkBaselines(s.testContexts[testutils.UnrestrictedReadWriteCtx], nbs))
		var (
			ids         []string
			cluster1ids []string
		)
		for _, nb := range nbs {
			ids = append(ids, nb.GetDeploymentId())
			if nb.GetClusterId() == testconsts.Cluster1 {
				cluster1ids = append(cluster1ids, nb.GetDeploymentId())
			}
		}
		s.testNBIDs = append(s.testNBIDs, ids...)
		// Try to delete everything.
		err := s.datastore.DeleteNetworkBaselines(s.testContexts[testutils.Cluster1ReadWriteCtx], ids)
		s.ErrorIs(err, sac.ErrResourceAccessDenied)
		// Check that nothing has been deleted.
		for _, id := range ids {
			_, ok, err := s.datastore.GetNetworkBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], id)
			s.NoError(err)
			s.True(ok)
		}

		// Try to delete only cluster1 resources.
		err = s.datastore.DeleteNetworkBaselines(s.testContexts[testutils.Cluster1ReadWriteCtx], cluster1ids)
		s.NoErrorf(err, "Must be able to delete all the %v resources", testconsts.Cluster1)
		// Check that all cluster1 resources have been deleted.
		for _, nb := range nbs {
			result, ok, err := s.datastore.GetNetworkBaseline(s.testContexts[testutils.UnrestrictedReadWriteCtx], nb.GetDeploymentId())
			s.NoError(err)
			if ok {
				s.NotEqual(testconsts.Cluster1, result.GetClusterId(), "The resource of %v must have been deleted", result.GetClusterId())
			} else {
				s.Equalf(testconsts.Cluster1, nb.GetClusterId(), "The resource of %v must have not been deleted", nb.GetClusterId())
			}
		}
	})
}
