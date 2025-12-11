//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	pgStore "github.com/stackrox/rox/central/serviceaccount/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestServiceAccountDataStore(t *testing.T) {
	suite.Run(t, new(ServiceAccountDataStoreTestSuite))
}

type ServiceAccountDataStoreTestSuite struct {
	suite.Suite

	pool      postgres.DB
	storage   store.Store
	datastore DataStore

	ctx context.Context
}

func (suite *ServiceAccountDataStoreTestSuite) SetupSuite() {
	pgtestbase := pgtest.ForT(suite.T())
	suite.Require().NotNil(pgtestbase)
	suite.pool = pgtestbase.DB
	suite.storage = pgStore.New(suite.pool)
	suite.datastore = New(suite.storage)

	suite.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ServiceAccount)))
}

func (suite *ServiceAccountDataStoreTestSuite) TearDownSuite() {
	suite.pool.Close()
}

func (suite *ServiceAccountDataStoreTestSuite) assertSearchResults(q *v1.Query, s *storage.ServiceAccount) {
	results, err := suite.datastore.SearchServiceAccounts(suite.ctx, q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Require().Len(results, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(results, 0)
	}
}

func (suite *ServiceAccountDataStoreTestSuite) TestServiceAccountsDataStore() {
	sa := fixtures.GetServiceAccount()
	err := suite.datastore.UpsertServiceAccount(suite.ctx, sa)
	suite.Require().NoError(err)

	foundSA, found, err := suite.datastore.GetServiceAccount(suite.ctx, sa.GetId())
	suite.Require().NoError(err)
	suite.True(found)
	protoassert.Equal(suite.T(), sa, foundSA)

	nonexistentID := uuid.Nil.String()
	_, found, err = suite.datastore.GetServiceAccount(suite.ctx, nonexistentID)
	suite.Require().NoError(err)
	suite.False(found)

	validQ := search.NewQueryBuilder().AddStrings(search.Cluster, sa.GetClusterName()).ProtoQuery()
	suite.assertSearchResults(validQ, sa)

	invalidQ := search.NewQueryBuilder().AddStrings(search.Cluster, nonexistentID).ProtoQuery()
	suite.assertSearchResults(invalidQ, nil)

	err = suite.datastore.RemoveServiceAccount(suite.ctx, sa.GetId())
	suite.Require().NoError(err)

	_, found, err = suite.datastore.GetServiceAccount(suite.ctx, sa.GetId())
	suite.Require().NoError(err)
	suite.False(found)

	suite.assertSearchResults(validQ, nil)
}

func (suite *ServiceAccountDataStoreTestSuite) TestSearchServiceAccounts() {
	// Create test service accounts
	sa1 := fixtures.GetServiceAccount()
	sa1.Id = uuid.NewV4().String()
	sa1.Name = "test-sa-1"
	sa1.ClusterName = "cluster-1"
	sa1.Namespace = "namespace-1"
	err := suite.datastore.UpsertServiceAccount(suite.ctx, sa1)
	suite.Require().NoError(err)

	sa2 := fixtures.GetServiceAccount()
	sa2.Id = uuid.NewV4().String()
	sa2.Name = "test-sa-2"
	sa2.ClusterName = "cluster-2"
	sa2.Namespace = "namespace-2"
	err = suite.datastore.UpsertServiceAccount(suite.ctx, sa2)
	suite.Require().NoError(err)

	sa3 := fixtures.GetServiceAccount()
	sa3.Id = uuid.NewV4().String()
	sa3.Name = "test-sa-3"
	sa3.ClusterName = "cluster-1"
	sa3.Namespace = "namespace-3"
	err = suite.datastore.UpsertServiceAccount(suite.ctx, sa3)
	suite.Require().NoError(err)

	// Define test cases
	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "empty query returns all service accounts with names populated",
			query:         search.EmptyQuery(),
			expectedCount: 3,
			expectedIDs:   []string{sa1.GetId(), sa2.GetId(), sa3.GetId()},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			expectedIDs:   []string{sa1.GetId(), sa2.GetId(), sa3.GetId()},
		},
		{
			name:          "query by service account name - exact match",
			query:         search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "test-sa-1").ProtoQuery(),
			expectedCount: 1,
			expectedIDs:   []string{sa1.GetId()},
		},
		{
			name:          "query by cluster name - multiple matches",
			query:         search.NewQueryBuilder().AddExactMatches(search.Cluster, "cluster-1").ProtoQuery(),
			expectedCount: 2,
			expectedIDs:   []string{sa1.GetId(), sa3.GetId()},
		},
		{
			name:          "query by namespace - single match",
			query:         search.NewQueryBuilder().AddExactMatches(search.Namespace, "namespace-2").ProtoQuery(),
			expectedCount: 1,
			expectedIDs:   []string{sa2.GetId()},
		},
		{
			name:          "query by service account ID",
			query:         search.NewQueryBuilder().AddExactMatches(search.ServiceAccountUID, sa3.GetId()).ProtoQuery(),
			expectedCount: 1,
			expectedIDs:   []string{sa3.GetId()},
		},
		{
			name:          "query with no matches returns empty",
			query:         search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, "nonexistent-sa").ProtoQuery(),
			expectedCount: 0,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			results, err := suite.datastore.SearchServiceAccounts(suite.ctx, tc.query)
			suite.NoError(err)
			suite.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			actualIDs := make([]string, 0, len(results))
			for _, result := range results {
				actualIDs = append(actualIDs, result.GetId())
				// Verify name is populated
				suite.NotEmpty(result.GetName(), "Name should be populated for service accounts")
				suite.Equal(v1.SearchCategory_SERVICE_ACCOUNTS, result.GetCategory())
				suite.Empty(result.GetLocation(), "Location should be empty for service accounts")
			}

			if len(tc.expectedIDs) > 0 {
				suite.ElementsMatch(tc.expectedIDs, actualIDs)
			}
		})
	}

	// Clean up
	suite.NoError(suite.datastore.RemoveServiceAccount(suite.ctx, sa1.GetId()))
	suite.NoError(suite.datastore.RemoveServiceAccount(suite.ctx, sa2.GetId()))
	suite.NoError(suite.datastore.RemoveServiceAccount(suite.ctx, sa3.GetId()))

	// Verify cleanup
	results, err := suite.datastore.SearchServiceAccounts(suite.ctx, search.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}
