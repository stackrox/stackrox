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

// TestSearchServiceAccounts_NameAndNilQuery verifies that SearchServiceAccounts populates the Name field
// and works correctly with a nil query as well as an exact name match query.
func (suite *ServiceAccountDataStoreTestSuite) TestSearchServiceAccounts_NameAndNilQuery() {
	// Insert two distinct service accounts
	sa1 := fixtures.GetServiceAccount()
	sa1.Name = "sa-1"
	sa2 := fixtures.GetServiceAccount()
	sa2.Id = uuid.NewV4().String()
	sa2.Name = "sa-2"

	for _, sa := range []*storage.ServiceAccount{sa1, sa2} {
		suite.Require().NoError(suite.datastore.UpsertServiceAccount(suite.ctx, sa))
		// Cleanup after test
		suite.T().Cleanup(func() {
			_ = suite.datastore.RemoveServiceAccount(suite.ctx, sa.GetId())
		})
	}

	// 1. Nil query should return both service accounts with populated Name fields
	results, err := suite.datastore.SearchServiceAccounts(suite.ctx, nil)
	suite.Require().NoError(err)
	ids := make(map[string]struct{})
	var actualNames []string
	for _, r := range results {
		if r.GetId() == sa1.GetId() || r.GetId() == sa2.GetId() {
			ids[r.GetId()] = struct{}{}
			suite.NotEmpty(r.GetName())
		}
		actualNames = append(actualNames, r.GetName())
	}
	suite.Contains(ids, sa1.GetId())
	suite.Contains(ids, sa2.GetId())
	suite.ElementsMatch([]string{sa1.GetName(), sa2.GetName()}, actualNames)

	// 2. Exact name query should return the matching service account
	nameQ := search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, sa1.GetName()).ProtoQuery()
	nameResults, err := suite.datastore.SearchServiceAccounts(suite.ctx, nameQ)
	suite.Require().NoError(err)
	suite.Len(nameResults, 1)
	suite.Equal(sa1.GetId(), nameResults[0].GetId())
	suite.Equal(sa1.GetName(), nameResults[0].GetName())
}
