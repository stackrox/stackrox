//go:build sql_integration

package datastore

import (
	"context"
	"testing"

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

func TestSecretDataStore(t *testing.T) {
	suite.Run(t, new(SecretDataStoreTestSuite))
}

type SecretDataStoreTestSuite struct {
	suite.Suite

	datastore DataStore

	pool postgres.DB

	ctx context.Context
}

func (suite *SecretDataStoreTestSuite) SetupSuite() {
	pgtestbase := pgtest.ForT(suite.T())
	suite.Require().NotNil(pgtestbase)
	suite.pool = pgtestbase.DB
	suite.datastore = GetTestPostgresDataStore(suite.T(), suite.pool)

	suite.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Secret)))
}

func (suite *SecretDataStoreTestSuite) TearDownSuite() {
	suite.pool.Close()
}

func (suite *SecretDataStoreTestSuite) assertSearchResults(q *v1.Query, s *storage.Secret) {
	results, err := suite.datastore.SearchSecrets(suite.ctx, q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Len(results, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(results, 0)
	}

	secrets, err := suite.datastore.SearchListSecrets(suite.ctx, q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Len(secrets, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(secrets, 0)
	}

	rawSecrets, err := suite.datastore.SearchRawSecrets(suite.ctx, q)
	suite.Require().NoError(err)
	if s != nil {
		suite.Len(rawSecrets, 1)
		suite.Equal(s.GetId(), results[0].GetId())
	} else {
		suite.Len(rawSecrets, 0)
	}
}

func (suite *SecretDataStoreTestSuite) TestSecretsDataStore() {
	secret := fixtures.GetSecret()
	err := suite.datastore.UpsertSecret(suite.ctx, secret)
	suite.Require().NoError(err)

	foundSecret, found, err := suite.datastore.GetSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)
	suite.True(found)
	protoassert.Equal(suite.T(), secret, foundSecret)

	nonExistentID := uuid.NewV4().String()
	_, found, err = suite.datastore.GetSecret(suite.ctx, nonExistentID)
	suite.Require().NoError(err)
	suite.False(found)

	validQ := search.NewQueryBuilder().AddStrings(search.Cluster, secret.GetClusterName()).ProtoQuery()
	suite.assertSearchResults(validQ, secret)

	invalidQ := search.NewQueryBuilder().AddStrings(search.Cluster, nonExistentID).ProtoQuery()
	suite.assertSearchResults(invalidQ, nil)

	err = suite.datastore.RemoveSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)

	_, found, err = suite.datastore.GetSecret(suite.ctx, secret.GetId())
	suite.Require().NoError(err)
	suite.False(found)

	suite.assertSearchResults(validQ, nil)
}

func (suite *SecretDataStoreTestSuite) TestSearchSecrets() {
	// Create test secrets
	secret1 := fixtures.GetSecret()
	secret1.Id = uuid.NewV4().String()
	secret1.Name = "test-secret-1"
	secret1.ClusterName = "cluster-1"
	secret1.Namespace = "namespace-1"
	err := suite.datastore.UpsertSecret(suite.ctx, secret1)
	suite.Require().NoError(err)

	secret2 := fixtures.GetSecret()
	secret2.Id = uuid.NewV4().String()
	secret2.Name = "test-secret-2"
	secret2.ClusterName = "cluster-2"
	secret2.Namespace = "namespace-2"
	err = suite.datastore.UpsertSecret(suite.ctx, secret2)
	suite.Require().NoError(err)

	secret3 := fixtures.GetSecret()
	secret3.Id = uuid.NewV4().String()
	secret3.Name = "test-secret-3"
	secret3.ClusterName = "cluster-1"
	secret3.Namespace = "namespace-3"
	err = suite.datastore.UpsertSecret(suite.ctx, secret3)
	suite.Require().NoError(err)

	// Define test cases
	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		expectedIDs   []string
		expectedNames []string
	}{
		{
			name:          "empty query returns all secrets with names populated",
			query:         search.EmptyQuery(),
			expectedCount: 3,
			expectedIDs:   []string{secret1.GetId(), secret2.GetId(), secret3.GetId()},
			expectedNames: []string{"test-secret-1", "test-secret-2", "test-secret-3"},
		},
		{
			name:          "nil query defaults to empty query",
			query:         nil,
			expectedCount: 3,
			expectedIDs:   []string{secret1.GetId(), secret2.GetId(), secret3.GetId()},
			expectedNames: []string{"test-secret-1", "test-secret-2", "test-secret-3"},
		},
		{
			name:          "query by secret name - exact match",
			query:         search.NewQueryBuilder().AddExactMatches(search.SecretName, "test-secret-1").ProtoQuery(),
			expectedCount: 1,
			expectedIDs:   []string{secret1.GetId()},
			expectedNames: []string{"test-secret-1"},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			results, err := suite.datastore.SearchSecrets(suite.ctx, tc.query)
			suite.NoError(err)
			suite.Len(results, tc.expectedCount, "Expected %d results, got %d", tc.expectedCount, len(results))

			actualIDs := make([]string, 0, len(results))
			actualNames := make([]string, 0, len(results))
			for _, result := range results {
				actualIDs = append(actualIDs, result.GetId())
				suite.Equal(v1.SearchCategory_SECRETS, result.GetCategory())
				actualNames = append(actualNames, result.GetName())
			}

			if len(tc.expectedIDs) > 0 {
				suite.ElementsMatch(tc.expectedIDs, actualIDs)
				suite.ElementsMatch(tc.expectedNames, actualNames)
			}
		})
	}

	// Clean up
	suite.NoError(suite.datastore.RemoveSecret(suite.ctx, secret1.GetId()))
	suite.NoError(suite.datastore.RemoveSecret(suite.ctx, secret2.GetId()))
	suite.NoError(suite.datastore.RemoveSecret(suite.ctx, secret3.GetId()))

	// Verify cleanup
	results, err := suite.datastore.SearchSecrets(suite.ctx, search.EmptyQuery())
	suite.NoError(err)
	suite.Empty(results)
}
