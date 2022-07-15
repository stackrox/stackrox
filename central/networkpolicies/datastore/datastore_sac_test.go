package store

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	boltStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/bolt"
	pgdbStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/postgres"
	undodeploymentstoremock "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/mocks"
	undostoremock "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestNetworkPolicySAC(t *testing.T) {
	suite.Run(t, new(networkPolicySACSuite))
}

type networkPolicySACSuite struct {
	suite.Suite

	datastore DataStore

	pool *pgxpool.Pool

	engine *bolt.DB

	storage store.Store

	testContexts         map[string]context.Context
	testNetworkPolicyIDs []string
}

func (s *networkPolicySACSuite) SetupSuite() {
	var err error
	if features.PostgresDatastore.Enabled() {
		ctx := context.Background()
		src := pgtest.GetConnectionString(s.T())
		cfg, err := pgxpool.ParseConfig(src)
		s.Require().NoError(err)
		s.pool, err = pgxpool.ConnectConfig(ctx, cfg)
		s.Require().NoError(err)
		pgdbStore.Destroy(ctx, s.pool)
		gormDB := pgtest.OpenGormDB(s.T(), src)
		defer pgtest.CloseGormDB(s.T(), gormDB)
		s.storage = pgdbStore.CreateTableAndNewStore(ctx, s.pool, gormDB)
	} else {
		s.engine, err = bolthelper.NewTemp(s.T().Name() + ".db")
		s.Require().NoError(err)
		s.storage = boltStore.New(s.engine)
	}
	mockCtrl := gomock.NewController(s.T())
	undomock := undostoremock.NewMockUndoStore(mockCtrl)
	undodeploymentmock := undodeploymentstoremock.NewMockUndoDeploymentStore(mockCtrl)

	s.datastore = New(s.storage, undomock, undodeploymentmock)

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkPolicy)
}

func (s *networkPolicySACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.Require().NoError(s.engine.Close())
	}
}

func (s *networkPolicySACSuite) SetupTest() {
	s.testNetworkPolicyIDs = make([]string, 0)
}

func (s *networkPolicySACSuite) TearDownTest() {
	for _, id := range s.testNetworkPolicyIDs {
		s.deleteNetworkPolicy(id)
	}
}

func (s *networkPolicySACSuite) deleteNetworkPolicy(id string) {
	s.Require().NoError(s.datastore.RemoveNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *networkPolicySACSuite) TestGetNetworkPolicy() {
	networkPolicy := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err := s.datastore.UpsertNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], networkPolicy)
	s.Require().NoError(err)
	s.testNetworkPolicyIDs = append(s.testNetworkPolicyIDs, networkPolicy.GetId())
	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			policy, found, err := s.datastore.GetNetworkPolicy(ctx, networkPolicy.GetId())
			s.NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Equal(networkPolicy, policy)
			} else {
				s.False(found)
				s.Nil(policy)
			}
		})
	}
}

func (s *networkPolicySACSuite) TestGetNetworkPolicies() {
	var err error
	networkPolicy1 := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err = s.datastore.UpsertNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], networkPolicy1)
	s.Require().NoError(err)
	s.testNetworkPolicyIDs = append(s.testNetworkPolicyIDs, networkPolicy1.GetId())
	networkPolicy2 := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster3, testconsts.NamespaceB)
	err = s.datastore.UpsertNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], networkPolicy2)
	s.Require().NoError(err)
	s.testNetworkPolicyIDs = append(s.testNetworkPolicyIDs, networkPolicy2.GetId())
	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			policies, err := s.datastore.GetNetworkPolicies(ctx, testconsts.Cluster2, testconsts.NamespaceB)
			s.NoError(err)
			if c.ExpectedFound {
				s.ElementsMatch([]*storage.NetworkPolicy{networkPolicy1}, policies)
			} else {
				s.ElementsMatch([]*storage.NetworkPolicy{}, policies)
			}
		})
	}
}

func (s *networkPolicySACSuite) TestCountMatchingNetworkPolicies() {
	var err error
	networkPolicy1 := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err = s.datastore.UpsertNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], networkPolicy1)
	s.Require().NoError(err)
	s.testNetworkPolicyIDs = append(s.testNetworkPolicyIDs, networkPolicy1.GetId())
	networkPolicy2 := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster3, testconsts.NamespaceB)
	err = s.datastore.UpsertNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], networkPolicy2)
	s.Require().NoError(err)
	s.testNetworkPolicyIDs = append(s.testNetworkPolicyIDs, networkPolicy2.GetId())
	cases := map[string]struct {
		scopeKey      string
		expectedCount int
	}{
		"global read-only can count": {
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedCount: 1,
		},
		"global read-write can count": {
			scopeKey:      testutils.UnrestrictedReadWriteCtx,
			expectedCount: 1,
		},
		"read-write on wrong cluster cannot count": {
			scopeKey: testutils.Cluster1ReadWriteCtx,
		},
		"read-write on wrong cluster and namespace cannot count": {
			scopeKey: testutils.Cluster1NamespaceAReadWriteCtx,
		},
		"read-write on wrong cluster and matching namespace cannot count": {
			scopeKey: testutils.Cluster1NamespaceBReadWriteCtx,
		},
		"read-write on matching cluster can count": {
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedCount: 1,
		},
		"read-write on matching cluster but wrong namespace cannot count": {
			scopeKey: testutils.Cluster2NamespaceAReadWriteCtx,
		},
		"read-write on matching cluster and namespace can count": {
			scopeKey:      testutils.Cluster2NamespaceBReadWriteCtx,
			expectedCount: 1,
		},
		"read-write on matching cluster and at least one matching namespace can count": {
			scopeKey:      testutils.Cluster2NamespacesABReadWriteCtx,
			expectedCount: 1,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			count, err := s.datastore.CountMatchingNetworkPolicies(ctx, testconsts.Cluster2, testconsts.NamespaceB)
			s.NoError(err)
			s.Equal(c.expectedCount, count)
		})
	}
}

func (s *networkPolicySACSuite) TestUpsertNetworkPolicy() {
	testedVerb := "upsert"
	cases := testutils.GenericNamespaceSACUpsertTestCases(s.T(), testedVerb)

	for name, c := range cases {
		s.Run(name, func() {
			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			ctx := s.testContexts[c.ScopeKey]
			policy := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			err := s.datastore.UpsertNetworkPolicy(ctx, policy)
			defer s.deleteNetworkPolicy(policy.GetId())
			if c.ExpectError {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			} else {
				s.NoError(err)
				count, countErr := s.datastore.CountMatchingNetworkPolicies(unrestrictedCtx, testconsts.Cluster2, testconsts.NamespaceB)
				s.NoError(countErr)
				s.Equal(1, count)
			}
		})
	}
}

func (s *networkPolicySACSuite) TestRemoveNetworkPolicy() {
	cases := testutils.GenericNamespaceSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			unrestrictedCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
			ctx := s.testContexts[c.ScopeKey]
			policy := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			err := s.datastore.UpsertNetworkPolicy(unrestrictedCtx, policy)
			s.Require().NoError(err)
			deleteErr := s.datastore.RemoveNetworkPolicy(ctx, policy.GetId())
			defer s.deleteNetworkPolicy(policy.GetId())
			if c.ExpectError {
				s.ErrorIs(deleteErr, sac.ErrResourceAccessDenied)
				count, countErr := s.datastore.CountMatchingNetworkPolicies(unrestrictedCtx, testconsts.Cluster2, testconsts.NamespaceB)
				s.NoError(countErr)
				s.Equal(1, count)
			} else {
				s.NoError(deleteErr)
				count, countErr := s.datastore.CountMatchingNetworkPolicies(unrestrictedCtx, testconsts.Cluster2, testconsts.NamespaceB)
				s.NoError(countErr)
				s.Empty(count)
			}
		})
	}
}
