package store

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	boltStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/bolt"
	pgdbStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore"
	undodeploymentstoremock "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/mocks"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	undostoremock "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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

	storage             store.Store
	undoStore           undostore.UndoStore
	undoDeploymentStore undodeploymentstore.UndoDeploymentStore

	mockCtrl *gomock.Controller

	testContexts         map[string]context.Context
	testNetworkPolicyIDs []string
}

func (s *networkPolicySACSuite) SetupSuite() {
	var err error
	if features.PostgresDatastore.Enabled() {
		ctx := context.Background()
		src := pgtest.GetConnectionString(s.T())
		cfg, err := pgxpool.ParseConfig(src)
		s.NoError(err)
		s.pool, err = pgxpool.ConnectConfig(ctx, cfg)
		s.NoError(err)
		pgdbStore.Destroy(ctx, s.pool)
		s.storage = pgdbStore.New(ctx, s.pool)
	} else {
		s.engine, err = bolthelper.NewTemp(s.T().Name() + ".db")
		s.NoError(err)
		s.storage = boltStore.New(s.engine)
	}
	s.mockCtrl = gomock.NewController(s.T())
	undomock := undostoremock.NewMockUndoStore(s.mockCtrl)
	undodeploymentmock := undodeploymentstoremock.NewMockUndoDeploymentStore(s.mockCtrl)

	s.datastore = New(s.storage, undomock, undodeploymentmock)

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.NetworkPolicy)
}

func (s *networkPolicySACSuite) TearDownSuite() {
	if features.PostgresDatastore.Enabled() {
		s.pool.Close()
	} else {
		s.NoError(s.engine.Close())
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
	s.NoError(s.datastore.RemoveNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], id))
}

func (s *networkPolicySACSuite) TestGetNetworkPolicy() {
	networkPolicy := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err := s.datastore.UpsertNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], networkPolicy)
	s.NoError(err)
	s.testNetworkPolicyIDs = append(s.testNetworkPolicyIDs, networkPolicy.GetId())
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
		"read-write on wrong cluster and namespace cannot get": {
			scopeKey: testutils.Cluster1NamespaceAReadWriteCtx,
		},
		"read-write on wrong cluster and matching namespace cannot get": {
			scopeKey: testutils.Cluster1NamespaceBReadWriteCtx,
		},
		"read-write on matching cluster can get": {
			scopeKey: testutils.Cluster2ReadWriteCtx,
			found:    true,
		},
		"read-write on matching cluster but wrong namespace cannot get": {
			scopeKey: testutils.Cluster2NamespaceAReadWriteCtx,
		},
		"read-write on matching cluster and namespace can get": {
			scopeKey: testutils.Cluster2NamespaceBReadWriteCtx,
			found:    true,
		},
		"read-write on matching cluster and at least one matching namespace can get": {
			scopeKey: testutils.Cluster2NamespacesABReadWriteCtx,
			found:    true,
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			policy, found, err := s.datastore.GetNetworkPolicy(ctx, networkPolicy.GetId())
			if c.found {
				s.True(found)
				s.NoError(err)
				s.Equal(networkPolicy, policy)
			} else {
				s.False(found)
				s.NoError(err)
				s.Nil(policy)
			}
		})
	}
}

func (s *networkPolicySACSuite) TestGetNetworkPolicies() {
	var err error
	networkPolicy1 := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err = s.datastore.UpsertNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], networkPolicy1)
	s.NoError(err)
	s.testNetworkPolicyIDs = append(s.testNetworkPolicyIDs, networkPolicy1.GetId())
	networkPolicy2 := fixtures.GetScopedNetworkPolicy(uuid.NewV4().String(), testconsts.Cluster3, testconsts.NamespaceB)
	err = s.datastore.UpsertNetworkPolicy(s.testContexts[testutils.UnrestrictedReadWriteCtx], networkPolicy2)
	s.NoError(err)
	s.testNetworkPolicyIDs = append(s.testNetworkPolicyIDs, networkPolicy2.GetId())
	cases := map[string]struct {
		scopeKey      string
		expectedErr   error
		expectedFound []*storage.NetworkPolicy
	}{
		"global read-only can get": {
			scopeKey:      testutils.UnrestrictedReadCtx,
			expectedFound: []*storage.NetworkPolicy{networkPolicy1},
		},
		"global read-write can get": {
			scopeKey:      testutils.UnrestrictedReadWriteCtx,
			expectedFound: []*storage.NetworkPolicy{networkPolicy1},
		},
		"read-write on wrong cluster cannot get": {
			scopeKey: testutils.Cluster1ReadWriteCtx,
		},
		"read-write on wrong cluster and namespace cannot get": {
			scopeKey: testutils.Cluster1NamespaceAReadWriteCtx,
		},
		"read-write on wrong cluster and matching namespace cannot get": {
			scopeKey: testutils.Cluster1NamespaceBReadWriteCtx,
		},
		"read-write on matching cluster can get": {
			scopeKey:      testutils.Cluster2ReadWriteCtx,
			expectedFound: []*storage.NetworkPolicy{networkPolicy1},
		},
		"read-write on matching cluster but wrong namespace cannot get": {
			scopeKey: testutils.Cluster2NamespaceAReadWriteCtx,
		},
		"read-write on matching cluster and namespace can get (partial)": {
			scopeKey:      testutils.Cluster2NamespaceBReadWriteCtx,
			expectedFound: []*storage.NetworkPolicy{networkPolicy1},
		},
		"read-write on matching cluster and at least one matching namespace can get (partial)": {
			scopeKey:      testutils.Cluster2NamespacesABReadWriteCtx,
			expectedFound: []*storage.NetworkPolicy{networkPolicy1},
		},
	}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.scopeKey]
			policies, err := s.datastore.GetNetworkPolicies(ctx, testconsts.Cluster2, testconsts.NamespaceB)
			s.ElementsMatch(c.expectedFound, policies)
			s.ErrorIs(err, c.expectedErr)
		})
	}
}
